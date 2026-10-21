import React, { useState } from 'react';
import type { Category, Feed } from '../types';
import { api } from '../lib/api';
import { formatDateRel } from '../lib/utils';
import {
  Plus,
  RefreshCw,
  Trash2,
  ExternalLink,
  Upload,
  Download,
  CheckCircle2,
  AlertCircle,
  FileText,
  Sparkles,
} from 'lucide-react';
import { toast } from 'sonner';

interface FeedsViewProps {
  feeds: Feed[];
  categories: Category[];
  onReload: () => void;
}

export const FeedsView: React.FC<FeedsViewProps> = ({
  feeds,
  categories,
  onReload,
}) => {
  const [showAddModal, setShowAddModal] = useState(false);
  const [feedUrl, setFeedUrl] = useState('');
  const [isValidating, setIsValidating] = useState(false);
  const [preview, setPreview] = useState<any>(null);

  // Formulaire d'ajout
  const [title, setTitle] = useState('');
  const [selectedCat, setSelectedCat] = useState<number | ''>('');
  const [fullText, setFullText] = useState(false);
  const [inDigest, setInDigest] = useState(true);
  const [isSaving, setIsSaving] = useState(false);

  // Validation d'URL
  const handleValidate = async () => {
    if (!feedUrl) return;
    setIsValidating(true);
    setPreview(null);
    try {
      const res = await api.validateFeed(feedUrl);
      setPreview(res);
      setTitle(res.title);
    } catch (err: any) {
      toast.error(err.message || 'Flux non valide');
    } finally {
      setIsValidating(false);
    }
  };

  const handleSaveFeed = async () => {
    if (!feedUrl) return;
    setIsSaving(true);
    try {
      await api.createFeed({
        url: preview ? preview.feed_url : feedUrl,
        title: title || (preview ? preview.title : ''),
        category_id: selectedCat === '' ? null : Number(selectedCat),
        full_text: fullText,
        include_in_digest: inDigest,
      });
      toast.success('Flux ajouté avec succès !');
      setShowAddModal(false);
      setFeedUrl('');
      setPreview(null);
      onReload();
    } catch (err: any) {
      toast.error('Erreur ajout flux: ' + err.message);
    } finally {
      setIsSaving(false);
    }
  };

  const handleDeleteFeed = async (id: number, title: string) => {
    if (!confirm(`Supprimer le flux "${title}" et tous ses articles ?`)) return;
    try {
      await api.deleteFeed(id);
      toast.success('Flux supprimé');
      onReload();
    } catch {
      toast.error('Erreur suppression flux');
    }
  };

  const handleRefreshFeed = async (id: number) => {
    try {
      const res = await api.refreshFeed(id);
      toast.success(`${res.new_articles} nouveaux articles récupérés`);
      onReload();
    } catch (err: any) {
      toast.error('Échec rafraîchissement: ' + err.message);
    }
  };

  const handleOPMLUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    try {
      const res = await api.importOPML(file);
      toast.success(res.message);
      onReload();
    } catch (err: any) {
      toast.error('Erreur import OPML: ' + err.message);
    }
  };

  return (
    <div className="flex-1 h-full overflow-y-auto p-6 md:p-8 bg-stone-50 dark:bg-stone-950">
      <div className="max-w-5xl mx-auto space-y-6">
        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-6 border-b border-stone-200 dark:border-stone-800">
          <div>
            <h1 className="text-2xl font-bold text-stone-900 dark:text-stone-100">
              Gestion des flux RSS & OPML
            </h1>
            <p className="text-sm text-stone-500">
              Configurez vos abonnements, l'extraction de texte intégral et l'inclusion dans le digest quotidien.
            </p>
          </div>

          <div className="flex items-center gap-2">
            {/* Export OPML */}
            <a
              href="/api/opml/export"
              download
              className="px-3 py-2 text-xs font-medium bg-stone-100 dark:bg-stone-800 hover:bg-stone-200 dark:hover:bg-stone-700 text-stone-700 dark:text-stone-300 rounded-lg flex items-center gap-1.5 transition-colors"
            >
              <Download className="w-4 h-4" />
              Exporter OPML
            </a>

            {/* Import OPML */}
            <label className="px-3 py-2 text-xs font-medium bg-stone-100 dark:bg-stone-800 hover:bg-stone-200 dark:hover:bg-stone-700 text-stone-700 dark:text-stone-300 rounded-lg flex items-center gap-1.5 cursor-pointer transition-colors">
              <Upload className="w-4 h-4" />
              Importer OPML
              <input
                type="file"
                accept=".xml,.opml"
                onChange={handleOPMLUpload}
                className="hidden"
              />
            </label>

            {/* Bouton Ajouter */}
            <button
              onClick={() => setShowAddModal(true)}
              className="px-4 py-2 text-xs font-semibold bg-amber-600 hover:bg-amber-700 text-white rounded-lg flex items-center gap-1.5 shadow-xs transition-colors"
            >
              <Plus className="w-4 h-4" />
              Ajouter un flux
            </button>
          </div>
        </div>

        {/* Tableau des flux */}
        <div className="bg-white dark:bg-stone-900 rounded-xl border border-stone-200 dark:border-stone-800 overflow-hidden shadow-xs">
          <table className="w-full text-left text-xs">
            <thead className="bg-stone-50 dark:bg-stone-800/60 text-stone-500 font-semibold uppercase tracking-wider border-b border-stone-200 dark:border-stone-800">
              <tr>
                <th className="p-3.5">Flux</th>
                <th className="p-3.5">Catégorie</th>
                <th className="p-3.5 text-center">Digest</th>
                <th className="p-3.5 text-center">Texte intégral</th>
                <th className="p-3.5">Articles</th>
                <th className="p-3.5">Dernière maj</th>
                <th className="p-3.5 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-stone-100 dark:divide-stone-800/60 text-stone-700 dark:text-stone-300">
              {feeds.length === 0 ? (
                <tr>
                  <td colSpan={7} className="p-8 text-center text-stone-400">
                    Aucun flux RSS configuré. Cliquez sur "Ajouter un flux" ou importez un fichier OPML.
                  </td>
                </tr>
              ) : (
                feeds.map((f) => {
                  const cat = categories.find((c) => c.id === f.category_id);
                  return (
                    <tr
                      key={f.id}
                      className="hover:bg-stone-50/80 dark:hover:bg-stone-800/40 transition-colors"
                    >
                      <td className="p-3.5 font-medium text-stone-900 dark:text-stone-100">
                        <div className="flex items-center gap-2">
                          <span className="truncate max-w-xs">{f.title}</span>
                          {f.site_url && (
                            <a
                              href={f.site_url}
                              target="_blank"
                              rel="noreferrer"
                              className="text-stone-400 hover:text-amber-600"
                            >
                              <ExternalLink className="w-3.5 h-3.5" />
                            </a>
                          )}
                        </div>
                        {f.last_error && (
                          <span className="text-[10px] text-red-500 flex items-center gap-1 mt-0.5">
                            <AlertCircle className="w-3 h-3" />
                            {f.last_error}
                          </span>
                        )}
                      </td>
                      <td className="p-3.5 text-stone-500">{cat?.name || '—'}</td>
                      <td className="p-3.5 text-center">
                        <span
                          className={`inline-block w-2.5 h-2.5 rounded-full ${
                            f.include_in_digest ? 'bg-emerald-500' : 'bg-stone-300 dark:bg-stone-700'
                          }`}
                        />
                      </td>
                      <td className="p-3.5 text-center">
                        <span
                          className={`inline-block w-2.5 h-2.5 rounded-full ${
                            f.full_text ? 'bg-amber-500' : 'bg-stone-300 dark:bg-stone-700'
                          }`}
                        />
                      </td>
                      <td className="p-3.5 font-mono">{f.article_count}</td>
                      <td className="p-3.5 text-stone-500">
                        {f.last_fetched_at ? formatDateRel(f.last_fetched_at) : 'Jamais'}
                      </td>
                      <td className="p-3.5 text-right space-x-1">
                        <button
                          onClick={() => handleRefreshFeed(f.id)}
                          className="p-1.5 text-stone-400 hover:text-stone-700 dark:hover:text-stone-200 rounded hover:bg-stone-100 dark:hover:bg-stone-800"
                          title="Rafraîchir ce flux maintenant"
                        >
                          <RefreshCw className="w-3.5 h-3.5" />
                        </button>
                        <button
                          onClick={() => handleDeleteFeed(f.id, f.title)}
                          className="p-1.5 text-stone-400 hover:text-red-600 rounded hover:bg-stone-100 dark:hover:bg-stone-800"
                          title="Supprimer ce flux"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Modale d'ajout de flux avec prévisualisation */}
      {showAddModal && (
        <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-white dark:bg-stone-900 border border-stone-200 dark:border-stone-800 rounded-2xl max-w-lg w-full p-6 space-y-4 shadow-xl">
            <h3 className="font-bold text-lg text-stone-900 dark:text-stone-100">
              Ajouter un flux RSS
            </h3>

            {/* Saisie URL */}
            <div className="space-y-1">
              <label className="text-xs font-semibold text-stone-600 dark:text-stone-400">
                URL du flux RSS ou du site web
              </label>
              <div className="flex gap-2">
                <input
                  type="text"
                  placeholder="https://example.com/feed.xml"
                  value={feedUrl}
                  onChange={(e) => setFeedUrl(e.target.value)}
                  className="flex-1 px-3 py-2 text-sm rounded-lg border border-stone-300 dark:border-stone-700 bg-stone-50 dark:bg-stone-800 text-stone-900 dark:text-stone-100 focus:outline-hidden focus:ring-2 focus:ring-amber-500"
                />
                <button
                  disabled={isValidating || !feedUrl}
                  onClick={handleValidate}
                  className="px-4 py-2 bg-stone-800 hover:bg-stone-700 dark:bg-stone-700 dark:hover:bg-stone-600 text-white text-xs font-medium rounded-lg disabled:opacity-50"
                >
                  {isValidating ? 'Analyse...' : 'Tester'}
                </button>
              </div>
            </div>

            {/* Aperçu validé */}
            {preview && (
              <div className="p-3 bg-amber-50/70 dark:bg-amber-950/30 border border-amber-200/60 dark:border-amber-900/60 rounded-xl space-y-2 text-xs">
                <div className="flex items-center gap-1.5 text-amber-800 dark:text-amber-300 font-semibold">
                  <CheckCircle2 className="w-4 h-4" />
                  <span>Flux détecté ({preview.article_count} articles)</span>
                </div>
                <div className="space-y-1 text-stone-600 dark:text-stone-400">
                  {preview.samples?.slice(0, 3).map((s: any, idx: number) => (
                    <div key={idx} className="truncate">• {s.title}</div>
                  ))}
                </div>
              </div>
            )}

            {/* Options */}
            <div className="space-y-3 pt-2">
              <div>
                <label className="block text-xs font-semibold text-stone-600 dark:text-stone-400 mb-1">
                  Titre personnalisé (optionnel)
                </label>
                <input
                  type="text"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder={preview?.title || 'Titre du flux'}
                  className="w-full px-3 py-2 text-sm rounded-lg border border-stone-300 dark:border-stone-700 bg-stone-50 dark:bg-stone-800 text-stone-900 dark:text-stone-100"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-stone-600 dark:text-stone-400 mb-1">
                  Catégorie
                </label>
                <select
                  value={selectedCat}
                  onChange={(e) => setSelectedCat(e.target.value === '' ? '' : Number(e.target.value))}
                  className="w-full px-3 py-2 text-sm rounded-lg border border-stone-300 dark:border-stone-700 bg-stone-50 dark:bg-stone-800 text-stone-900 dark:text-stone-100"
                >
                  <option value="">Aucune catégorie</option>
                  {categories.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.name}
                    </option>
                  ))}
                </select>
              </div>

              <div className="flex flex-col gap-2 pt-1 text-xs">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={inDigest}
                    onChange={(e) => setInDigest(e.target.checked)}
                    className="rounded text-amber-600"
                  />
                  <span>Inclure automatiquement dans le digest quotidien</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={fullText}
                    onChange={(e) => setFullText(e.target.checked)}
                    className="rounded text-amber-600"
                  />
                  <span>Extraire le texte intégral par défaut (Readability)</span>
                </label>
              </div>
            </div>

            {/* Boutons actions */}
            <div className="flex items-center justify-end gap-2 pt-4 border-t border-stone-200 dark:border-stone-800">
              <button
                onClick={() => setShowAddModal(false)}
                className="px-4 py-2 text-xs font-medium text-stone-600 dark:text-stone-400 hover:bg-stone-100 dark:hover:bg-stone-800 rounded-lg"
              >
                Annuler
              </button>
              <button
                disabled={isSaving || !feedUrl}
                onClick={handleSaveFeed}
                className="px-4 py-2 text-xs font-semibold bg-amber-600 hover:bg-amber-700 text-white rounded-lg disabled:opacity-50"
              >
                {isSaving ? 'Enregistrement...' : 'Enregistrer'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
