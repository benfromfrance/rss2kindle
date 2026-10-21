import React, { createContext, useContext, useEffect, useState } from 'react';
import type { Article } from '../types';
import { api } from '../lib/api';
import { toast } from 'sonner';

interface CartContextType {
  items: Article[];
  isOpen: boolean;
  openCart: () => void;
  closeCart: () => void;
  toggleCart: () => void;
  addItem: (article: Article) => void;
  removeItem: (articleId: number) => void;
  toggleItem: (article: Article) => void;
  isInCart: (articleId: number) => boolean;
  clearCart: () => void;
  moveItem: (index: number, direction: 'up' | 'down') => void;
  sendToKindle: (title: string) => Promise<void>;
  downloadEpub: (title: string) => Promise<void>;
  saveAsCollection: (name: string) => Promise<void>;
  isSending: boolean;
}

const CartContext = createContext<CartContextType | undefined>(undefined);

const STORAGE_KEY = 'rss2kindle_cart_items';

export const CartProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [items, setItems] = useState<Article[]>(() => {
    try {
      const saved = localStorage.getItem(STORAGE_KEY);
      return saved ? JSON.parse(saved) : [];
    } catch {
      return [];
    }
  });
  const [isOpen, setIsOpen] = useState(false);
  const [isSending, setIsSending] = useState(false);

  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(items));
    } catch {
      // ignore
    }
  }, [items]);

  const addItem = (article: Article) => {
    setItems((prev) => {
      if (prev.some((a) => a.id === article.id)) return prev;
      return [...prev, article];
    });
    toast.success('Article ajouté au panier Kindle');
  };

  const removeItem = (articleId: number) => {
    setItems((prev) => prev.filter((a) => a.id !== articleId));
  };

  const toggleItem = (article: Article) => {
    setItems((prev) => {
      const exists = prev.some((a) => a.id === article.id);
      if (exists) {
        toast.info('Article retiré du panier Kindle');
        return prev.filter((a) => a.id !== article.id);
      } else {
        toast.success('Article ajouté au panier Kindle');
        return [...prev, article];
      }
    });
  };

  const isInCart = (articleId: number) => items.some((a) => a.id === articleId);

  const clearCart = () => setItems([]);

  const moveItem = (index: number, direction: 'up' | 'down') => {
    setItems((prev) => {
      const newItems = [...prev];
      const targetIndex = direction === 'up' ? index - 1 : index + 1;
      if (targetIndex < 0 || targetIndex >= newItems.length) return prev;
      const temp = newItems[index];
      newItems[index] = newItems[targetIndex];
      newItems[targetIndex] = temp;
      return newItems;
    });
  };

  const sendToKindle = async (title: string) => {
    if (items.length === 0) return;
    setIsSending(true);
    try {
      await api.createDelivery({
        kind: 'selection',
        title: title || 'Sélection Kindle',
        article_ids: items.map((a) => a.id),
        send: true,
      });
      toast.success('Livraison envoyée avec succès au Kindle !');
      clearCart();
      setIsOpen(false);
    } catch (err: any) {
      toast.error('Échec de la livraison: ' + err.message);
    } finally {
      setIsSending(false);
    }
  };

  const downloadEpub = async (title: string) => {
    if (items.length === 0) return;
    setIsSending(true);
    try {
      const deliv = await api.createDelivery({
        kind: 'selection',
        title: title || 'Recueil Kindle',
        article_ids: items.map((a) => a.id),
        send: false,
      });

      // Téléchargement automatique dès que prêt
      window.location.href = `/api/deliveries/${deliv.id}/epub`;
      toast.success('Téléchargement de votre recueil EPUB démarré');
      setIsOpen(false);
    } catch (err: any) {
      toast.error('Échec de génération EPUB: ' + err.message);
    } finally {
      setIsSending(false);
    }
  };

  const saveAsCollection = async (name: string) => {
    if (items.length === 0) return;
    try {
      const col = await api.createCollection(name);
      await api.collectionItems(
        col.id,
        'add',
        items.map((a) => a.id)
      );
      toast.success(`Collection "${name}" créée avec ${items.length} articles`);
    } catch (err: any) {
      toast.error('Erreur création collection: ' + err.message);
    }
  };

  return (
    <CartContext.Provider
      value={{
        items,
        isOpen,
        openCart: () => setIsOpen(true),
        closeCart: () => setIsOpen(false),
        toggleCart: () => setIsOpen((prev) => !prev),
        addItem,
        removeItem,
        toggleItem,
        isInCart,
        clearCart,
        moveItem,
        sendToKindle,
        downloadEpub,
        saveAsCollection,
        isSending,
      }}
    >
      {children}
    </CartContext.Provider>
  );
};

export const useCart = () => {
  const ctx = useContext(CartContext);
  if (!ctx) throw new Error('useCart must be used within a CartProvider');
  return ctx;
};
