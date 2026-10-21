import React, { useEffect, useState } from 'react';
import type { Collection } from '../types';
import { api } from '../lib/api';
import { formatDateFull } from '../lib/utils';
import { Folder, Send, Download, Trash2, Plus, BookOpen } from 'lucide-react';
import { toast } from 'sonner';

interface CollectionsViewProps {
  onSelectCollection: (colId: number, name: string) => void;
}

export const CollectionsView: React.FC<CollectionsViewProps> = ({ onSelectCollection }) => {
  const [collections, setCollections] = useState<Collection[]>([]);
  const [newColName, setNewColName] = useState('');
  const [isCreating, setIsCreating] = useState(false);

  const load = () => {
    api.listCollections().then(setCollections).catch(() => {});
  };

  useEffect(() => {
    load();
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newColName.trim()) return;
    setIsCreating(true);
    try {
      await api.createCollection(newColName.trim());
      setNewColName('');
      toast.success('Collection créée');
      load();
    } catch (err: any) {
      toast.error('Erreur: ' + err.message);
    } finally {
      setIsCreating(false);
    }
  };

  const handleDelete = async (id: number, name: string) => {
    if (!confirm(`Supprimer la collection "${name}" ?`)) return;
    try {
      await api.deleteCollection(id);
      toast.success('Collection supprimée');
      load();
    } catch {
      toast.error('Erreur suppression');
    }
  };

  const handleSendCollection = async (col: Collection) => {
    try {
      await api.createDelivery({
        kind: 'collection',
        collection_id: col.id,
        send: true,
      });
      toast.success(`Livraison de "${col.name}" envoyée vers le Kindle !`);
    } catch (err: any) {
      toast.error('Erreur envoi: ' + err.message);
    }
  };

  return (
    <div className="flex-1 h-full overflow-y-auto p-6 md:p-8 bg-stone-50 dark:bg-stone-950">
      <div className="max-w-4xl mx-auto space-y-6">
        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-6 border-b border-stone-200 dark:border-stone-800">
          <div>
            <h1 className="text-2xl font-bold text-stone-900 dark:text-stone-100">
              Mes Collections
            </h1>
            <p className="text-sm text-stone-500">
              Dossiers personnalisés d'articles que vous pouvez composer et envoyer au Kindle en un clic.
            </p>
          </div>
        </div>

        {/* Formulaire nouvelle collection */}
        <form
          onSubmit={handleCreate}
          className="bg-white dark:bg-stone-900 p-4 rounded-xl border border-stone-200 dark:border-stone-800 flex gap-2 shadow-xs"
        >
          <input
            type="text"
            required
            placeholder="Nom de la nouvelle collection (ex: À lire ce weekend)..."
            value={newColName}
            onChange={(e) => setNewColName(e.target.value)}
            className="flex-1 px-3 py-2 text-sm rounded-lg border border-stone-300 dark:border-stone-700 bg-stone-50 dark:bg-stone-800 text-stone-900 dark:text-stone-100"
          />
          <button
            type="submit"
            disabled={isCreating}
            className="px-4 py-2 text-xs font-semibold bg-amber-600 hover:bg-amber-700 text-white rounded-lg flex items-center gap-1.5 transition-colors disabled:opacity-50"
          >
            <Plus className="w-4 h-4" />
            Créer
          </button>
        </form>

        {/* Liste des collections */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {collections.length === 0 ? (
            <div className="col-span-2 py-12 text-center text-stone-400">
              Aucune collection. Créez-en une ci-dessus ou sauvegardez votre panier Kindle comme collection.
            </div>
          ) : (
            collections.map((col) => (
              <div
                key={col.id}
                className="bg-white dark:bg-stone-900 p-5 rounded-xl border border-stone-200 dark:border-stone-800 flex flex-col justify-between space-y-4 shadow-xs hover:border-amber-500/50 transition-colors"
              >
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-2.5">
                    <div className="p-2 bg-amber-50 dark:bg-amber-950/40 text-amber-700 dark:text-amber-400 rounded-lg">
                      <Folder className="w-5 h-5" />
                    </div>
                    <div>
                      <h3
                        onClick={() => onSelectCollection(col.id, col.name)}
                        className="font-semibold text-stone-900 dark:text-stone-100 cursor-pointer hover:underline"
                      >
                        {col.name}
                      </h3>
                      <span className="text-xs text-stone-500">
                        {col.article_count} {col.article_count === 1 ? 'article' : 'articles'}
                      </span>
                    </div>
                  </div>

                  <button
                    onClick={() => handleDelete(col.id, col.name)}
                    className="p-1.5 text-stone-400 hover:text-red-500 rounded"
                    title="Supprimer la collection"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>

                <div className="pt-2 border-t border-stone-100 dark:border-stone-800 flex items-center justify-between text-xs">
                  <button
                    onClick={() => onSelectCollection(col.id, col.name)}
                    className="text-stone-600 dark:text-stone-400 hover:text-stone-900 dark:hover:text-stone-100 font-medium"
                  >
                    Consulter les articles →
                  </button>

                  <button
                    disabled={col.article_count === 0}
                    onClick={() => handleSendCollection(col)}
                    className="px-3 py-1.5 bg-amber-600 hover:bg-amber-700 text-white rounded-lg flex items-center gap-1 font-medium disabled:opacity-40"
                  >
                    <Send className="w-3.5 h-3.5" />
                    Envoyer au Kindle
                  </button>
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
};
