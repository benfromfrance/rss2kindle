import React, { useEffect, useState, useCallback } from 'react';
import type { Article, ArticlePage } from '../types';
import { api } from '../lib/api';
import { useCart } from '../context/CartContext';
import { ArticleDetail } from './ArticleDetail';
import { formatDateRel } from '../lib/utils';
import {
  CheckCheck,
  Search,
  Star,
  Check,
  Plus,
  BookOpen,
  Send,
  Sparkles,
  Inbox,
  Filter,
} from 'lucide-react';
import { toast } from 'sonner';

interface ReaderViewProps {
  view: string;
  feedId?: number;
  categoryId?: number;
  collectionId?: number;
  title: string;
}

export const ReaderView: React.FC<ReaderViewProps> = ({
  view,
  feedId,
  categoryId,
  collectionId,
  title,
}) => {
  const { isInCart, toggleItem } = useCart();

  const [articles, setArticles] = useState<Article[]>([]);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [nextCursor, setNextCursor] = useState<string>('');
  const [isLoading, setIsLoading] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [activeQuery, setActiveQuery] = useState('');

  // Chargement des articles
  const loadArticles = useCallback(
    async (cursor = '') => {
      setIsLoading(true);
      try {
        const res = await api.listArticles({
          view,
          feed_id: feedId,
          category_id: categoryId,
          collection_id: collectionId,
          q: activeQuery,
          cursor,
          limit: 30,
        });

        if (cursor) {
          setArticles((prev) => [...prev, ...res.items]);
        } else {
          setArticles(res.items);
          if (res.items.length > 0 && !selectedId && window.innerWidth >= 768) {
            setSelectedId(res.items[0].id);
          }
        }
        setNextCursor(res.next_cursor || '');
      } catch (err: any) {
        toast.error('Erreur chargement articles: ' + err.message);
      } finally {
        setIsLoading(false);
      }
    },
    [view, feedId, categoryId, collectionId, activeQuery]
  );

  useEffect(() => {
    loadArticles('');
  }, [loadArticles]);

  const selectedArticle = articles.find((a) => a.id === selectedId) || null;

  // Mise à jour d'un article dans la liste
  const handleArticleUpdate = (updated: Article) => {
    setArticles((prev) => prev.map((a) => (a.id === updated.id ? updated : a)));
  };

  const handleMarkAllRead = async () => {
    try {
      const res = await api.markAllRead(feedId, categoryId);
      setArticles((prev) => prev.map((a) => ({ ...a, is_read: true })));
      toast.success(`${res.affected} articles marqués comme lus`);
    } catch {
      toast.error('Erreur marquer tout comme lu');
    }
  };

  // Raccourcis clavier (j/k, s, m, x)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Ignorer si on tape dans un champ de texte
      if (['INPUT', 'TEXTAREA'].includes((e.target as HTMLElement)?.tagName)) {
        return;
      }

      const currentIndex = articles.findIndex((a) => a.id === selectedId);

      if (e.key === 'j') {
        // Suivant
        if (currentIndex < articles.length - 1) {
          const next = articles[currentIndex + 1];
          setSelectedId(next.id);
          if (!next.is_read) {
            api.patchArticle(next.id, { is_read: true });
            handleArticleUpdate({ ...next, is_read: true });
          }
        }
      } else if (e.key === 'k') {
        // Précédent
        if (currentIndex > 0) {
          const prev = articles[currentIndex - 1];
          setSelectedId(prev.id);
        }
      } else if (e.key === 's' && selectedArticle) {
        // Star
        const next = !selectedArticle.is_starred;
        api.patchArticle(selectedArticle.id, { is_starred: next });
        handleArticleUpdate({ ...selectedArticle, is_starred: next });
      } else if (e.key === 'm' && selectedArticle) {
        // Toggle lu
        const next = !selectedArticle.is_read;
        api.patchArticle(selectedArticle.id, { is_read: next });
        handleArticleUpdate({ ...selectedArticle, is_read: next });
      } else if (e.key === 'x' && selectedArticle) {
        // Panier Kindle
        toggleItem(selectedArticle);
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [articles, selectedId, selectedArticle, toggleItem]);

  return (
    <div className="h-full flex flex-col md:flex-row overflow-hidden bg-stone-50 dark:bg-stone-950">
      {/* Colonne centrale : Liste des articles */}
      <div
        className={`w-full md:w-96 lg:w-[420px] h-full flex flex-col border-r border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900 shrink-0 ${
          selectedId && 'hidden md:flex'
        }`}
      >
        {/* En-tête de la liste */}
        <div className="p-3 border-b border-stone-200 dark:border-stone-800 space-y-2">
          <div className="flex items-center justify-between">
            <h2 className="font-semibold text-stone-900 dark:text-stone-100 truncate text-base">
              {title}
            </h2>
            <button
              onClick={handleMarkAllRead}
              className="text-xs text-stone-500 hover:text-stone-800 dark:hover:text-stone-200 flex items-center gap-1 px-2 py-1 rounded hover:bg-stone-100 dark:hover:bg-stone-800"
              title="Tout marquer comme lu"
            >
              <CheckCheck className="w-3.5 h-3.5" />
              <span>Tout lu</span>
            </button>
          </div>

          {/* Recherche FTS5 */}
          <form
            onSubmit={(e) => {
              e.preventDefault();
              setActiveQuery(searchQuery);
            }}
            className="relative"
          >
            <Search className="w-4 h-4 absolute left-2.5 top-2.5 text-stone-400" />
            <input
              type="text"
              placeholder="Rechercher dans les articles..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg border border-stone-200 dark:border-stone-700 bg-stone-50 dark:bg-stone-800/60 text-stone-900 dark:text-stone-100 focus:outline-hidden focus:ring-2 focus:ring-amber-500"
            />
          </form>
        </div>

        {/* Liste défilante des articles */}
        <div className="flex-1 overflow-y-auto divide-y divide-stone-100 dark:divide-stone-800/60">
          {articles.length === 0 && !isLoading && (
            <div className="py-20 text-center text-stone-400 space-y-2">
              <Inbox className="w-10 h-10 mx-auto stroke-1" />
              <p className="text-sm">Aucun article dans cette vue</p>
            </div>
          )}

          {articles.map((art) => {
            const isSelected = art.id === selectedId;
            const inCart = isInCart(art.id);

            return (
              <div
                key={art.id}
                onClick={() => {
                  setSelectedId(art.id);
                  if (!art.is_read) {
                    api.patchArticle(art.id, { is_read: true });
                    handleArticleUpdate({ ...art, is_read: true });
                  }
                }}
                className={`p-3 cursor-pointer transition-colors relative flex items-start gap-2.5 group ${
                  isSelected
                    ? 'bg-amber-50/70 dark:bg-amber-950/30'
                    : 'hover:bg-stone-50 dark:hover:bg-stone-800/40'
                } ${!art.is_read ? 'font-semibold' : 'text-stone-600 dark:text-stone-400'}`}
              >
                {/* Indicateur non lu */}
                <span
                  className={`w-2 h-2 rounded-full mt-1.5 shrink-0 transition-opacity ${
                    !art.is_read ? 'bg-amber-600' : 'opacity-0'
                  }`}
                />

                <div className="flex-1 min-w-0 space-y-1">
                  <div className="flex items-center justify-between text-[11px] text-stone-400 dark:text-stone-500">
                    <span className="truncate font-medium text-amber-700 dark:text-amber-500/90">
                      {art.feed_title}
                    </span>
                    <span className="shrink-0">{formatDateRel(art.published_at)}</span>
                  </div>

                  <h3
                    className={`text-sm line-clamp-2 leading-snug ${
                      isSelected
                        ? 'text-amber-900 dark:text-amber-200'
                        : 'text-stone-900 dark:text-stone-100'
                    }`}
                  >
                    {art.title}
                  </h3>

                  <p className="text-xs text-stone-500 dark:text-stone-400 line-clamp-2 leading-relaxed">
                    {art.summary}
                  </p>
                </div>

                {/* Actions au survol / badges */}
                <div className="flex flex-col items-center gap-1 shrink-0 pt-0.5">
                  {/* Bouton Panier Kindle */}
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      toggleItem(art);
                    }}
                    title={inCart ? 'Retirer du panier Kindle' : 'Ajouter au panier Kindle'}
                    className={`p-1 rounded transition-colors ${
                      inCart
                        ? 'text-amber-600 dark:text-amber-400 bg-amber-100 dark:bg-amber-950/60'
                        : 'text-stone-300 dark:text-stone-600 hover:text-amber-600 opacity-0 group-hover:opacity-100'
                    }`}
                  >
                    {inCart ? <Check className="w-3.5 h-3.5" /> : <Plus className="w-3.5 h-3.5" />}
                  </button>

                  {/* Étoile favori */}
                  {art.is_starred && (
                    <Star className="w-3.5 h-3.5 text-amber-500 fill-amber-500" />
                  )}
                </div>
              </div>
            );
          })}

          {/* Bouton charger plus */}
          {nextCursor && (
            <div className="p-3 text-center">
              <button
                disabled={isLoading}
                onClick={() => loadArticles(nextCursor)}
                className="text-xs text-stone-600 dark:text-stone-300 font-medium px-4 py-2 bg-stone-100 dark:bg-stone-800 hover:bg-stone-200 dark:hover:bg-stone-700 rounded-lg w-full transition-colors"
              >
                {isLoading ? 'Chargement...' : 'Charger plus d’articles'}
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Colonne de droite : Panneau de lecture du texte complet */}
      <div className={`flex-1 h-full ${!selectedId ? 'hidden md:flex' : 'flex'}`}>
        {selectedArticle ? (
          <ArticleDetail
            article={selectedArticle}
            onUpdate={handleArticleUpdate}
            onClose={() => setSelectedId(null)}
          />
        ) : (
          <div className="flex-1 flex flex-col items-center justify-center text-center p-8 text-stone-400 space-y-4">
            <BookOpen className="w-16 h-16 stroke-1 text-stone-300 dark:text-stone-700" />
            <div className="space-y-1">
              <h3 className="text-lg font-medium text-stone-700 dark:text-stone-300">
                Sélectionnez un article pour commencer la lecture
              </h3>
              <p className="text-sm text-stone-500">
                Utilisez les touches <kbd className="px-1.5 py-0.5 bg-stone-200 dark:bg-stone-800 rounded font-mono text-xs">j</kbd> et <kbd className="px-1.5 py-0.5 bg-stone-200 dark:bg-stone-800 rounded font-mono text-xs">k</kbd> pour naviguer rapidement.
              </p>
            </div>

            <div className="flex items-center gap-4 text-xs text-stone-500 pt-4 border-t border-stone-200 dark:border-stone-800">
              <span><kbd className="font-mono">s</kbd> Favori</span>
              <span><kbd className="font-mono">m</kbd> Lu/Non lu</span>
              <span><kbd className="font-mono">x</kbd> Ajouter au Kindle</span>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
