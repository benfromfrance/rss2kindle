import React, { useEffect, useState } from 'react';
import type { Delivery } from '../types';
import { api } from '../lib/api';
import { formatDateFull, formatBytes } from '../lib/utils';
import {
  Send,
  Download,
  RotateCw,
  Trash2,
  CheckCircle,
  AlertCircle,
  Clock,
  Sparkles,
  BookOpen,
} from 'lucide-react';
import { toast } from 'sonner';

export const DeliveriesView: React.FC = () => {
  const [deliveries, setDeliveries] = useState<Delivery[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isTriggering, setIsTriggering] = useState(false);

  const loadDeliveries = async () => {
    setIsLoading(true);
    try {
      const list = await api.listDeliveries(50);
      setDeliveries(list);
    } catch {
      toast.error('Erreur chargement livraisons');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadDeliveries();
    // Rafraîchissement automatique toutes les 5s si une livraison est en cours
    const interval = setInterval(() => {
      api.listDeliveries(20).then(setDeliveries).catch(() => {});
    }, 5000);
    return () => clearInterval(interval);
  }, []);

  const handleRunDigestNow = async () => {
    setIsTriggering(true);
    try {
      const res = await api.createDelivery({ kind: 'digest' });
      toast.success('Génération du Digest quotidien lancée !');
      loadDeliveries();
    } catch (err: any) {
      toast.error(err.message || 'Échec lancement digest');
    } finally {
      setIsTriggering(false);
    }
  };

  const handleResend = async (id: number, regenerate = false) => {
    try {
      await api.resendDelivery(id, regenerate);
      toast.success(regenerate ? 'Regénération du recueil lancée' : 'Renvoi immédiat programmé');
      loadDeliveries();
    } catch (err: any) {
      toast.error('Échec renvoi: ' + err.message);
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await api.deleteDelivery(id);
      setDeliveries((prev) => prev.filter((d) => d.id !== id));
      toast.success('Livraison supprimée');
    } catch {
      toast.error('Erreur suppression');
    }
  };

  const getStatusBadge = (status: Delivery['status']) => {
    switch (status) {
      case 'sent':
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-100 text-emerald-800 dark:bg-emerald-950/60 dark:text-emerald-300">
            <CheckCircle className="w-3 h-3" />
            Envoyé
          </span>
        );
      case 'ready':
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-950/60 dark:text-blue-300">
            <Download className="w-3 h-3" />
            Prêt (Téléchargement)
          </span>
        );
      case 'building':
      case 'sending':
      case 'queued':
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-amber-100 text-amber-800 dark:bg-amber-950/60 dark:text-amber-300 animate-pulse">
            <Clock className="w-3 h-3" />
            {status === 'building' ? 'Génération EPUB...' : status === 'sending' ? 'Envoi SMTP...' : 'En attente...'}
          </span>
        );
      case 'empty':
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-stone-200 text-stone-700 dark:bg-stone-800 dark:text-stone-300">
            Aucun article
          </span>
        );
      case 'failed':
      default:
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800 dark:bg-red-950/60 dark:text-red-300">
            <AlertCircle className="w-3 h-3" />
            Échec
          </span>
        );
    }
  };

  return (
    <div className="flex-1 h-full overflow-y-auto p-6 md:p-8 bg-stone-50 dark:bg-stone-950">
      <div className="max-w-5xl mx-auto space-y-6">
        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-6 border-b border-stone-200 dark:border-stone-800">
          <div>
            <h1 className="text-2xl font-bold text-stone-900 dark:text-stone-100">
              Livraisons & Livres Kindle
            </h1>
            <p className="text-sm text-stone-500">
              Historique des e-books générés, état des envois SMTP vers votre Kindle et téléchargements directs.
            </p>
          </div>

          <button
            disabled={isTriggering}
            onClick={handleRunDigestNow}
            className="px-4 py-2.5 text-xs font-semibold bg-amber-600 hover:bg-amber-700 text-white rounded-lg flex items-center gap-2 shadow-xs transition-colors disabled:opacity-50"
          >
            <Send className="w-4 h-4" />
            {isTriggering ? 'Lancement...' : 'Générer et envoyer le Digest maintenant'}
          </button>
        </div>

        {/* Tableau des livraisons */}
        <div className="bg-white dark:bg-stone-900 rounded-xl border border-stone-200 dark:border-stone-800 overflow-hidden shadow-xs">
          <table className="w-full text-left text-xs">
            <thead className="bg-stone-50 dark:bg-stone-800/60 text-stone-500 font-semibold uppercase tracking-wider border-b border-stone-200 dark:border-stone-800">
              <tr>
                <th className="p-3.5">Recueil / Titre</th>
                <th className="p-3.5">Type</th>
                <th className="p-3.5">Statut</th>
                <th className="p-3.5">Articles</th>
                <th className="p-3.5">Taille</th>
                <th className="p-3.5">Date</th>
                <th className="p-3.5 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-stone-100 dark:divide-stone-800/60 text-stone-700 dark:text-stone-300">
              {deliveries.length === 0 ? (
                <tr>
                  <td colSpan={7} className="p-8 text-center text-stone-400">
                    Aucune livraison pour le moment.
                  </td>
                </tr>
              ) : (
                deliveries.map((d) => (
                  <tr
                    key={d.id}
                    className="hover:bg-stone-50/80 dark:hover:bg-stone-800/40 transition-colors"
                  >
                    <td className="p-3.5 font-medium text-stone-900 dark:text-stone-100">
                      <div className="font-semibold">{d.title}</div>
                      {d.recipient && (
                        <div className="text-[11px] text-stone-400">Vers {d.recipient}</div>
                      )}
                      {d.error && (
                        <div className="text-[11px] text-red-500 flex items-center gap-1 mt-0.5">
                          <AlertCircle className="w-3 h-3" />
                          {d.error}
                        </div>
                      )}
                    </td>
                    <td className="p-3.5">
                      <span className="capitalize px-2 py-0.5 bg-stone-100 dark:bg-stone-800 rounded text-[11px]">
                        {d.kind}
                      </span>
                    </td>
                    <td className="p-3.5">{getStatusBadge(d.status)}</td>
                    <td className="p-3.5 font-mono">{d.article_count}</td>
                    <td className="p-3.5 font-mono text-stone-500">
                      {d.size_bytes > 0 ? formatBytes(d.size_bytes) : '—'}
                    </td>
                    <td className="p-3.5 text-stone-500">{formatDateFull(d.created_at)}</td>
                    <td className="p-3.5 text-right space-x-2">
                      {/* Télécharger EPUB */}
                      {d.has_epub && (
                        <a
                          href={`/api/deliveries/${d.id}/epub`}
                          download
                          className="inline-flex items-center p-1.5 text-stone-500 hover:text-amber-600 rounded hover:bg-stone-100 dark:hover:bg-stone-800"
                          title="Télécharger le fichier EPUB"
                        >
                          <Download className="w-3.5 h-3.5" />
                        </a>
                      )}

                      {/* Renvoyer / Regénérer */}
                      <button
                        onClick={() => handleResend(d.id, false)}
                        className="p-1.5 text-stone-500 hover:text-stone-900 dark:hover:text-stone-100 rounded hover:bg-stone-100 dark:hover:bg-stone-800"
                        title="Renvoyer ce livre au Kindle"
                      >
                        <RotateCw className="w-3.5 h-3.5" />
                      </button>

                      <button
                        onClick={() => handleDelete(d.id)}
                        className="p-1.5 text-stone-400 hover:text-red-500 rounded hover:bg-stone-100 dark:hover:bg-stone-800"
                        title="Supprimer"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};
