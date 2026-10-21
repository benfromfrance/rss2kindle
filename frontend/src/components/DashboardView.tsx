import React, { useEffect, useState } from 'react';
import type { StatusResponse } from '../types';
import { api } from '../lib/api';
import { formatDateFull, formatDateRel } from '../lib/utils';
import {
  Clock,
  Send,
  Rss,
  CheckCircle,
  Inbox,
  TrendingUp,
  Sparkles,
  BookOpen,
} from 'lucide-react';
import { toast } from 'sonner';

interface DashboardViewProps {
  onGoToReader: () => void;
  onGoToDeliveries: () => void;
}

export const DashboardView: React.FC<DashboardViewProps> = ({
  onGoToReader,
  onGoToDeliveries,
}) => {
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [isSending, setIsSending] = useState(false);

  const loadStatus = () => {
    api.getStatus().then(setStatus).catch(() => {});
  };

  useEffect(() => {
    loadStatus();
  }, []);

  const handleTriggerDigest = async () => {
    setIsSending(true);
    try {
      await api.createDelivery({ kind: 'digest' });
      toast.success('Digest quotidien généré et envoyé au Kindle !');
      loadStatus();
    } catch (err: any) {
      toast.error(err.message || 'Échec lancement digest');
    } finally {
      setIsSending(false);
    }
  };

  if (!status) {
    return <div className="p-8 text-center text-stone-500">Chargement du tableau de bord...</div>;
  }

  return (
    <div className="flex-1 h-full overflow-y-auto p-6 md:p-8 bg-stone-50 dark:bg-stone-950">
      <div className="max-w-5xl mx-auto space-y-6">
        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-6 border-b border-stone-200 dark:border-stone-800">
          <div>
            <h1 className="text-2xl font-bold text-stone-900 dark:text-stone-100">
              Tableau de bord
            </h1>
            <p className="text-sm text-stone-500">
              Vue d'ensemble de vos flux, livraisons automatiques et lectures en attente.
            </p>
          </div>

          <button
            disabled={isSending}
            onClick={handleTriggerDigest}
            className="px-4 py-2.5 text-xs font-semibold bg-amber-600 hover:bg-amber-700 text-white rounded-lg flex items-center gap-2 shadow-xs transition-colors disabled:opacity-50"
          >
            <Send className="w-4 h-4" />
            {isSending ? 'Envoi...' : 'Envoyer le Digest maintenant'}
          </button>
        </div>

        {/* 4 Blocs statistiques clés */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="bg-white dark:bg-stone-900 p-5 rounded-xl border border-stone-200 dark:border-stone-800 shadow-xs">
            <div className="flex items-center justify-between text-stone-500 text-xs font-medium">
              <span>Prochain Digest</span>
              <Clock className="w-4 h-4 text-amber-600" />
            </div>
            <div className="text-lg font-bold text-stone-900 dark:text-stone-100 mt-2">
              {status.next_digest_run ? formatDateRel(status.next_digest_run) : 'Non planifié'}
            </div>
            <div className="text-[11px] text-stone-400 mt-1">
              {status.next_digest_run ? formatDateFull(status.next_digest_run) : 'Désactivé'}
            </div>
          </div>

          <div
            onClick={onGoToReader}
            className="bg-white dark:bg-stone-900 p-5 rounded-xl border border-stone-200 dark:border-stone-800 shadow-xs cursor-pointer hover:border-amber-500/50 transition-colors"
          >
            <div className="flex items-center justify-between text-stone-500 text-xs font-medium">
              <span>Articles non lus</span>
              <Inbox className="w-4 h-4 text-emerald-600" />
            </div>
            <div className="text-2xl font-bold text-stone-900 dark:text-stone-100 mt-2">
              {status.counts?.unread || 0}
            </div>
            <div className="text-[11px] text-stone-400 mt-1">Sur {status.counts?.total || 0} au total</div>
          </div>

          <div className="bg-white dark:bg-stone-900 p-5 rounded-xl border border-stone-200 dark:border-stone-800 shadow-xs">
            <div className="flex items-center justify-between text-stone-500 text-xs font-medium">
              <span>En attente d'envoi Kindle</span>
              <Send className="w-4 h-4 text-amber-600" />
            </div>
            <div className="text-2xl font-bold text-stone-900 dark:text-stone-100 mt-2">
              {status.counts?.unsent || 0}
            </div>
            <div className="text-[11px] text-stone-400 mt-1">Seront inclus dans le prochain digest</div>
          </div>

          <div
            onClick={onGoToDeliveries}
            className="bg-white dark:bg-stone-900 p-5 rounded-xl border border-stone-200 dark:border-stone-800 shadow-xs cursor-pointer hover:border-amber-500/50 transition-colors"
          >
            <div className="flex items-center justify-between text-stone-500 text-xs font-medium">
              <span>Dernier Digest envoyé</span>
              <CheckCircle className="w-4 h-4 text-emerald-600" />
            </div>
            <div className="text-sm font-bold text-stone-900 dark:text-stone-100 mt-2 truncate">
              {status.last_digest ? status.last_digest.title : 'Aucun envoi récent'}
            </div>
            <div className="text-[11px] text-stone-400 mt-1">
              {status.last_digest ? formatDateRel(status.last_digest.created_at) : '—'}
            </div>
          </div>
        </div>

        {/* Activité sur 14 jours (Histogramme) */}
        <div className="bg-white dark:bg-stone-900 rounded-xl border border-stone-200 dark:border-stone-800 p-6 space-y-4 shadow-xs">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <TrendingUp className="w-5 h-5 text-amber-600" />
              <h2 className="font-semibold text-stone-900 dark:text-stone-100 text-sm">
                Activité sur les 14 derniers jours
              </h2>
            </div>
            <div className="flex items-center gap-4 text-xs">
              <span className="flex items-center gap-1.5 text-stone-500">
                <span className="w-2.5 h-2.5 bg-stone-300 dark:bg-stone-700 rounded-xs" />
                Collectés
              </span>
              <span className="flex items-center gap-1.5 text-stone-500">
                <span className="w-2.5 h-2.5 bg-amber-600 rounded-xs" />
                Envoyés au Kindle
              </span>
            </div>
          </div>

          {/* Barres */}
          <div className="h-44 flex items-end gap-2 pt-6">
            {status.stats?.map((st) => {
              const max = Math.max(
                ...status.stats.map((s) => Math.max(s.fetched, s.sent, 10))
              );
              const fetchedHeight = (st.fetched / max) * 100;
              const sentHeight = (st.sent / max) * 100;
              const dateLabel = st.day.slice(5); // MM-DD

              return (
                <div key={st.day} className="flex-1 flex flex-col items-center gap-1 h-full justify-end group">
                  <div className="w-full flex items-end justify-center gap-1 h-32">
                    {/* Fetched */}
                    <div
                      style={{ height: `${fetchedHeight}%` }}
                      className="w-full max-w-[12px] bg-stone-200 dark:bg-stone-800 rounded-t-xs transition-all group-hover:bg-stone-300 dark:group-hover:bg-stone-700"
                      title={`${st.fetched} articles collectés`}
                    />
                    {/* Sent */}
                    <div
                      style={{ height: `${sentHeight}%` }}
                      className="w-full max-w-[12px] bg-amber-600 rounded-t-xs transition-all group-hover:bg-amber-500"
                      title={`${st.sent} articles envoyés au Kindle`}
                    />
                  </div>
                  <span className="text-[10px] text-stone-400 font-mono mt-1">{dateLabel}</span>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
};
