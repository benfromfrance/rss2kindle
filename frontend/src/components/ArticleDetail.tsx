import React, { useState } from 'react';
import type { Article } from '../types';
import { api } from '../lib/api';
import { useCart } from '../context/CartContext';
import { formatDateFull } from '../lib/utils';
import {
  Star,
  CheckCircle,
  ExternalLink,
  BookOpen,
  Plus,
  Check,
  Sparkles,
  Type,
  Share2,
} from 'lucide-react';
import { toast } from 'sonner';

interface ArticleDetailProps {
  article: Article;
  onUpdate: (updated: Article) => void;
  onClose?: () => void;
}

export const ArticleDetail: React.FC<ArticleDetailProps> = ({
  article,
  onUpdate,
  onClose,
}) => {
  const { isInCart, toggleItem } = useCart();
  const [fontSize, setFontSize] = useState<'sm' | 'base' | 'lg' | 'xl'>('base');
  const [isExtracting, setIsExtracting] = useState(false);

  const inCart = isInCart(article.id);

  const toggleStar = async () => {
    const next = !article.is_starred;
    try {
      await api.patchArticle(article.id, { is_starred: next });
      onUpdate({ ...article, is_starred: next });
      toast.success(next ? 'Ajouté aux favoris' : 'Retiré des favoris');
    } catch {
      toast.error('Erreur mise à jour favori');
    }
  };

  const toggleRead = async () => {
    const next = !article.is_read;
    try {
      await api.patchArticle(article.id, { is_read: next });
      onUpdate({ ...article, is_read: next });
    } catch {
      toast.error('Erreur mise à jour statut lu');
    }
  };

  const loadFullText = async () => {
    setIsExtracting(true);
    try {
      const res = await api.extractFullText(article.id);
      onUpdate({
        ...article,
        full_content_html: res.full_content_html,
        has_full_content: true,
      });
      toast.success('Texte intégral extrait avec succès !');
    } catch (err: any) {
      toast.error('Échec extraction: ' + err.message);
    } finally {
      setIsExtracting(false);
    }
  };

  const contentToDisplay =
    article.full_content_html || article.content_html || `<p>${article.summary}</p>`;

  const fontSizeClasses = {
    sm: 'text-base',
    base: 'text-lg',
    lg: 'text-xl',
    xl: 'text-2xl',
  };

  return (
    <div className="h-full flex flex-col bg-stone-50 dark:bg-stone-900 border-l border-stone-200 dark:border-stone-800 overflow-hidden">
      {/* Barre d'outils du lecteur */}
      <div className="p-3 border-b border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900/90 backdrop-blur-xs flex items-center justify-between gap-2 shrink-0">
        <div className="flex items-center gap-1">
          {onClose && (
            <button
              onClick={onClose}
              className="md:hidden p-1.5 rounded-lg text-stone-500 hover:bg-stone-100 dark:hover:bg-stone-800"
            >
              ← Retour
            </button>
          )}

          {/* Toggle lu */}
          <button
            onClick={toggleRead}
            title={article.is_read ? 'Marquer non-lu' : 'Marquer comme lu'}
            className={`p-2 rounded-lg transition-colors ${
              article.is_read
                ? 'text-stone-400 hover:text-stone-600 dark:hover:text-stone-200'
                : 'text-emerald-600 dark:text-emerald-500 bg-emerald-50 dark:bg-emerald-950/40'
            }`}
          >
            <CheckCircle className="w-5 h-5" />
          </button>

          {/* Toggle favori */}
          <button
            onClick={toggleStar}
            title={article.is_starred ? 'Retirer des favoris' : 'Ajouter aux favoris'}
            className={`p-2 rounded-lg transition-colors ${
              article.is_starred
                ? 'text-amber-500 fill-amber-500'
                : 'text-stone-400 hover:text-amber-500'
            }`}
          >
            <Star
              className={`w-5 h-5 ${article.is_starred ? 'fill-amber-500' : ''}`}
            />
          </button>

          {/* Ajouter au panier Kindle */}
          <button
            onClick={() => toggleItem(article)}
            className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg transition-all ${
              inCart
                ? 'bg-amber-600 text-white shadow-xs'
                : 'bg-stone-100 dark:bg-stone-800 text-stone-700 dark:text-stone-300 hover:bg-amber-100 dark:hover:bg-amber-950/50 hover:text-amber-700'
            }`}
          >
            {inCart ? <Check className="w-4 h-4" /> : <Plus className="w-4 h-4" />}
            <span>Kindle</span>
          </button>
        </div>

        <div className="flex items-center gap-2">
          {/* Extraction Readability */}
          {!article.has_full_content && article.url && (
            <button
              disabled={isExtracting}
              onClick={loadFullText}
              className="flex items-center gap-1 px-2.5 py-1.5 text-xs font-medium rounded-lg bg-amber-50 dark:bg-amber-950/40 text-amber-700 dark:text-amber-400 hover:bg-amber-100 dark:hover:bg-amber-900/40 disabled:opacity-50"
              title="Télécharger la page originale et extraire l'article complet"
            >
              <Sparkles className="w-3.5 h-3.5" />
              <span>{isExtracting ? 'Extraction...' : 'Texte intégral'}</span>
            </button>
          )}

          {/* Réglage taille de texte */}
          <div className="flex items-center bg-stone-100 dark:bg-stone-800 rounded-lg p-0.5 text-xs text-stone-600 dark:text-stone-300">
            <button
              onClick={() => setFontSize('sm')}
              className={`px-2 py-1 rounded ${fontSize === 'sm' ? 'bg-white dark:bg-stone-700 font-bold shadow-xs' : ''}`}
            >
              A-
            </button>
            <button
              onClick={() => setFontSize('base')}
              className={`px-2 py-1 rounded ${fontSize === 'base' ? 'bg-white dark:bg-stone-700 font-bold shadow-xs' : ''}`}
            >
              A
            </button>
            <button
              onClick={() => setFontSize('lg')}
              className={`px-2 py-1 rounded ${fontSize === 'lg' ? 'bg-white dark:bg-stone-700 font-bold shadow-xs' : ''}`}
            >
              A+
            </button>
          </div>

          {/* Lien externe */}
          {article.url && (
            <a
              href={article.url}
              target="_blank"
              rel="noreferrer"
              className="p-2 text-stone-400 hover:text-stone-700 dark:hover:text-stone-200 rounded-lg hover:bg-stone-100 dark:hover:bg-stone-800"
              title="Ouvrir l'article original"
            >
              <ExternalLink className="w-4 h-4" />
            </a>
          )}
        </div>
      </div>

      {/* Conteneur défilant du texte de l'article */}
      <div className="flex-1 overflow-y-auto px-6 py-8 md:px-12 max-w-3xl mx-auto w-full">
        {/* Métadonnées de l'article */}
        <header className="mb-8 space-y-3 border-b border-stone-200 dark:border-stone-800 pb-6">
          <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-amber-700 dark:text-amber-400">
            <BookOpen className="w-4 h-4 inline" />
            <span>{article.feed_title}</span>
            {article.sent_at && (
              <span className="ml-auto normal-case bg-stone-200 dark:bg-stone-800 text-stone-700 dark:text-stone-300 px-2 py-0.5 rounded text-[11px]">
                Envoyé au Kindle
              </span>
            )}
          </div>

          <h1 className="text-2xl md:text-3xl font-bold font-serif tracking-tight text-stone-900 dark:text-stone-100 leading-snug">
            {article.title}
          </h1>

          <div className="flex flex-wrap items-center gap-3 text-xs text-stone-500 dark:text-stone-400">
            {article.author && <span>Par {article.author}</span>}
            {article.author && <span>•</span>}
            <span>{formatDateFull(article.published_at)}</span>
            {article.has_full_content && (
              <>
                <span>•</span>
                <span className="text-amber-600 dark:text-amber-400 font-medium">
                  Texte intégral extrait
                </span>
              </>
            )}
          </div>
        </header>

        {/* Corps de l'article */}
        <article
          className={`prose-reader ${fontSizeClasses[fontSize]} text-stone-800 dark:text-stone-200`}
          dangerouslySetInnerHTML={{ __html: contentToDisplay }}
        />

        {/* Pied de page */}
        <footer className="mt-12 pt-6 border-t border-stone-200 dark:border-stone-800 flex flex-col sm:flex-row items-center justify-between gap-4 text-xs text-stone-500">
          <div>Source originale : {article.url}</div>
          <div className="flex items-center gap-3">
            <button
              onClick={() => toggleItem(article)}
              className="px-4 py-2 bg-amber-600 hover:bg-amber-700 text-white font-medium rounded-lg flex items-center gap-1.5 transition-colors"
            >
              <Plus className="w-4 h-4" />
              {inCart ? 'Présent dans le panier Kindle' : 'Ajouter au recueil Kindle'}
            </button>
          </div>
        </footer>
      </div>
    </div>
  );
};
