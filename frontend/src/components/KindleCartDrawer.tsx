import React, { useState } from 'react';
import { useCart } from '../context/CartContext';
import {
  X,
  Send,
  Download,
  Trash2,
  ChevronUp,
  ChevronDown,
  BookmarkPlus,
  BookOpen,
} from 'lucide-react';

export const KindleCartDrawer: React.FC = () => {
  const {
    items,
    isOpen,
    closeCart,
    removeItem,
    clearCart,
    moveItem,
    sendToKindle,
    downloadEpub,
    saveAsCollection,
    isSending,
  } = useCart();

  const [title, setTitle] = useState('');
  const [collectionName, setCollectionName] = useState('');
  const [showSaveCol, setShowSaveCol] = useState(false);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 overflow-hidden bg-black/40 backdrop-blur-xs flex justify-end">
      <div className="w-full max-w-md bg-stone-50 dark:bg-stone-900 border-l border-stone-200 dark:border-stone-800 h-full flex flex-col shadow-2xl animate-in slide-in-from-right duration-200">
        {/* Header */}
        <div className="p-4 border-b border-stone-200 dark:border-stone-800 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <BookOpen className="w-5 h-5 text-amber-600 dark:text-amber-500" />
            <h2 className="font-semibold text-lg text-stone-900 dark:text-stone-100">
              Panier Kindle ({items.length})
            </h2>
          </div>
          <button
            onClick={closeCart}
            className="p-1 rounded-lg text-stone-500 hover:text-stone-900 dark:hover:text-stone-100 hover:bg-stone-200 dark:hover:bg-stone-800"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          {items.length === 0 ? (
            <div className="text-center py-16 text-stone-500 space-y-3">
              <BookOpen className="w-12 h-12 mx-auto stroke-1 text-stone-400" />
              <p className="font-medium">Votre panier Kindle est vide</p>
              <p className="text-sm">
                Sélectionnez des articles dans le lecteur pour composer un recueil sur mesure.
              </p>
            </div>
          ) : (
            <>
              {/* Configuration du titre du livre */}
              <div>
                <label className="block text-xs font-medium uppercase tracking-wider text-stone-500 dark:text-stone-400 mb-1">
                  Titre du recueil
                </label>
                <input
                  type="text"
                  placeholder="Ex: Sélection tech du weekend"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  className="w-full px-3 py-2 text-sm rounded-lg border border-stone-300 dark:border-stone-700 bg-white dark:bg-stone-800 text-stone-900 dark:text-stone-100 focus:outline-hidden focus:ring-2 focus:ring-amber-500"
                />
              </div>

              {/* Liste des articles avec réordonnancement */}
              <div className="space-y-2">
                <div className="flex justify-between items-center text-xs text-stone-500">
                  <span>Ordre des chapitres ({items.length})</span>
                  <button
                    onClick={clearCart}
                    className="text-red-500 hover:underline flex items-center gap-1"
                  >
                    <Trash2 className="w-3 h-3" />
                    Tout vider
                  </button>
                </div>

                <div className="space-y-2">
                  {items.map((art, idx) => (
                    <div
                      key={art.id}
                      className="p-3 bg-white dark:bg-stone-800/80 rounded-lg border border-stone-200 dark:border-stone-700/60 flex items-start gap-2 shadow-xs group"
                    >
                      <div className="flex flex-col text-stone-400">
                        <button
                          disabled={idx === 0}
                          onClick={() => moveItem(idx, 'up')}
                          className="hover:text-stone-800 dark:hover:text-stone-200 disabled:opacity-20"
                        >
                          <ChevronUp className="w-4 h-4" />
                        </button>
                        <button
                          disabled={idx === items.length - 1}
                          onClick={() => moveItem(idx, 'down')}
                          className="hover:text-stone-800 dark:hover:text-stone-200 disabled:opacity-20"
                        >
                          <ChevronDown className="w-4 h-4" />
                        </button>
                      </div>

                      <div className="flex-1 min-w-0">
                        <span className="text-xs font-medium text-amber-700 dark:text-amber-400 block truncate">
                          {art.feed_title}
                        </span>
                        <h4 className="text-sm font-medium text-stone-900 dark:text-stone-100 line-clamp-2">
                          {art.title}
                        </h4>
                      </div>

                      <button
                        onClick={() => removeItem(art.id)}
                        className="text-stone-400 hover:text-red-500 p-1 rounded"
                      >
                        <X className="w-4 h-4" />
                      </button>
                    </div>
                  ))}
                </div>
              </div>

              {/* Option sauvegarde en collection */}
              <div className="pt-2">
                {!showSaveCol ? (
                  <button
                    onClick={() => setShowSaveCol(true)}
                    className="text-xs text-stone-600 dark:text-stone-400 hover:text-stone-900 dark:hover:text-stone-100 flex items-center gap-1.5"
                  >
                    <BookmarkPlus className="w-4 h-4" />
                    Enregistrer cette sélection comme collection
                  </button>
                ) : (
                  <div className="flex gap-2">
                    <input
                      type="text"
                      placeholder="Nom de la collection..."
                      value={collectionName}
                      onChange={(e) => setCollectionName(e.target.value)}
                      className="flex-1 px-3 py-1.5 text-xs rounded-lg border border-stone-300 dark:border-stone-700 bg-white dark:bg-stone-800 text-stone-900 dark:text-stone-100"
                    />
                    <button
                      onClick={async () => {
                        if (collectionName) {
                          await saveAsCollection(collectionName);
                          setCollectionName('');
                          setShowSaveCol(false);
                        }
                      }}
                      className="px-3 py-1.5 bg-stone-200 dark:bg-stone-700 text-xs font-medium rounded-lg hover:bg-stone-300 dark:hover:bg-stone-600"
                    >
                      Sauver
                    </button>
                  </div>
                )}
              </div>
            </>
          )}
        </div>

        {/* Footer actions */}
        {items.length > 0 && (
          <div className="p-4 border-t border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900 space-y-2">
            <button
              disabled={isSending}
              onClick={() => sendToKindle(title)}
              className="w-full py-2.5 px-4 bg-amber-600 hover:bg-amber-700 text-white font-medium rounded-lg flex items-center justify-center gap-2 shadow-sm transition-colors disabled:opacity-50"
            >
              <Send className="w-4 h-4" />
              {isSending ? 'Envoi en cours...' : 'Envoyer au Kindle par email'}
            </button>

            <button
              disabled={isSending}
              onClick={() => downloadEpub(title)}
              className="w-full py-2 px-4 bg-stone-100 dark:bg-stone-800 hover:bg-stone-200 dark:hover:bg-stone-700 text-stone-800 dark:text-stone-200 font-medium rounded-lg flex items-center justify-center gap-2 text-sm transition-colors disabled:opacity-50"
            >
              <Download className="w-4 h-4" />
              Télécharger le fichier EPUB seul
            </button>
          </div>
        )}
      </div>
    </div>
  );
};
