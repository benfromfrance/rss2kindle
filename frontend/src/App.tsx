import React, { useEffect, useState, useCallback } from 'react';
import { AuthProvider, useAuth } from './context/AuthContext';
import { CartProvider, useCart } from './context/CartContext';
import { Toaster, toast } from 'sonner';
import { api } from './lib/api';
import type { Category, Counts, Feed, Collection } from './types';
import { Sidebar } from './components/Sidebar';
import { ReaderView } from './components/ReaderView';
import { FeedsView } from './components/FeedsView';
import { CollectionsView } from './components/CollectionsView';
import { DeliveriesView } from './components/DeliveriesView';
import { SettingsView } from './components/SettingsView';
import { DashboardView } from './components/DashboardView';
import { KindleCartDrawer } from './components/KindleCartDrawer';
import { LoginModal } from './components/LoginModal';
import { BookOpen, Menu, ShoppingBag } from 'lucide-react';

const MainApp: React.FC = () => {
  const { isAuthenticated, isLoading, logout } = useAuth();
  const { items: cartItems, openCart } = useCart();

  const [currentView, setCurrentView] = useState('unread');
  const [viewTitle, setViewTitle] = useState('Articles non lus');
  const [viewMeta, setViewMeta] = useState<{
    feedId?: number;
    categoryId?: number;
    collectionId?: number;
  }>({});

  const [counts, setCounts] = useState<Counts | null>(null);
  const [categories, setCategories] = useState<Category[]>([]);
  const [feeds, setFeeds] = useState<Feed[]>([]);
  const [collections, setCollections] = useState<Collection[]>([]);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [isMobileSidebarOpen, setIsMobileSidebarOpen] = useState(false);

  // Thème sombre
  const [isDark, setIsDark] = useState(() => {
    const saved = localStorage.getItem('theme');
    if (saved) return saved === 'dark';
    return window.matchMedia('(prefers-color-scheme: dark)').matches;
  });

  useEffect(() => {
    if (isDark) {
      document.documentElement.classList.add('dark');
      localStorage.setItem('theme', 'dark');
    } else {
      document.documentElement.classList.remove('dark');
      localStorage.setItem('theme', 'light');
    }
  }, [isDark]);

  // Chargement des données globales
  const loadGlobalData = useCallback(async () => {
    if (!isAuthenticated) return;
    try {
      const [cts, cats, fds, cols] = await Promise.all([
        api.getCounts(),
        api.listCategories(),
        api.listFeeds(),
        api.listCollections(),
      ]);
      setCounts(cts);
      setCategories(cats);
      setFeeds(fds);
      setCollections(cols);
    } catch {
      // ignore
    }
  }, [isAuthenticated]);

  useEffect(() => {
    loadGlobalData();
    const interval = setInterval(loadGlobalData, 30000);
    return () => clearInterval(interval);
  }, [loadGlobalData]);

  const handleSelectView = (
    view: string,
    meta?: { feedId?: number; categoryId?: number; collectionId?: number; title?: string }
  ) => {
    setCurrentView(view);
    if (meta?.title) setViewTitle(meta.title);
    setViewMeta({
      feedId: meta?.feedId,
      categoryId: meta?.categoryId,
      collectionId: meta?.collectionId,
    });
    setIsMobileSidebarOpen(false);
  };

  const handleRefreshAll = async () => {
    setIsRefreshing(true);
    try {
      await api.refreshAllFeeds();
      toast.success('Rafraîchissement des flux lancé');
      setTimeout(loadGlobalData, 3000);
    } catch {
      toast.error('Erreur rafraîchissement');
    } finally {
      setIsRefreshing(false);
    }
  };

  if (isLoading) {
    return (
      <div className="h-full flex items-center justify-center bg-stone-50 dark:bg-stone-950 text-stone-500">
        Chargement de RSS to Kindle...
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col overflow-hidden">
      {/* Top Bar Mobile / Global */}
      <header className="h-12 border-b border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900 px-4 flex items-center justify-between shrink-0">
        <div className="flex items-center gap-2">
          <button
            onClick={() => setIsMobileSidebarOpen(!isMobileSidebarOpen)}
            className="md:hidden p-1.5 rounded-lg text-stone-600 dark:text-stone-300 hover:bg-stone-100 dark:hover:bg-stone-800"
          >
            <Menu className="w-5 h-5" />
          </button>
          <span className="font-semibold text-sm text-stone-900 dark:text-stone-100 hidden sm:inline">
            {viewTitle}
          </span>
        </div>

        {/* Bouton Panier Kindle avec badge */}
        <button
          onClick={openCart}
          className="relative flex items-center gap-2 px-3 py-1.5 text-xs font-semibold rounded-lg bg-amber-50 dark:bg-amber-950/40 text-amber-900 dark:text-amber-200 hover:bg-amber-100 dark:hover:bg-amber-900/40 transition-colors border border-amber-200/60 dark:border-amber-900/60 shadow-2xs"
        >
          <ShoppingBag className="w-4 h-4 text-amber-600 dark:text-amber-400" />
          <span>Panier Kindle</span>
          {cartItems.length > 0 && (
            <span className="px-1.5 py-0.2 rounded-full bg-amber-600 text-white font-mono text-[10px] font-bold">
              {cartItems.length}
            </span>
          )}
        </button>
      </header>

      {/* Main Layout avec Sidebar */}
      <div className="flex-1 flex overflow-hidden relative">
        {/* Sidebar Desktop */}
        <div className="hidden md:block h-full">
          <Sidebar
            currentView={currentView}
            onSelectView={handleSelectView}
            counts={counts}
            categories={categories}
            feeds={feeds}
            collections={collections}
            onRefreshAll={handleRefreshAll}
            isRefreshing={isRefreshing}
            onLogout={logout}
            isDark={isDark}
            onToggleDark={() => setIsDark(!isDark)}
          />
        </div>

        {/* Sidebar Mobile Overlay */}
        {isMobileSidebarOpen && (
          <div className="fixed inset-0 z-40 md:hidden flex">
            <div
              onClick={() => setIsMobileSidebarOpen(false)}
              className="fixed inset-0 bg-black/40 backdrop-blur-xs"
            />
            <div className="relative z-50 h-full">
              <Sidebar
                currentView={currentView}
                onSelectView={handleSelectView}
                counts={counts}
                categories={categories}
                feeds={feeds}
                collections={collections}
                onRefreshAll={handleRefreshAll}
                isRefreshing={isRefreshing}
                onLogout={logout}
                isDark={isDark}
                onToggleDark={() => setIsDark(!isDark)}
              />
            </div>
          </div>
        )}

        {/* Vue active */}
        <main className="flex-1 h-full overflow-hidden flex flex-col">
          {currentView === 'feeds' ? (
            <FeedsView
              feeds={feeds}
              categories={categories}
              onReload={loadGlobalData}
            />
          ) : currentView === 'collections' ? (
            <CollectionsView
              onSelectCollection={(colId, name) =>
                handleSelectView(`col-${colId}`, {
                  collectionId: colId,
                  title: `Collection : ${name}`,
                })
              }
            />
          ) : currentView === 'deliveries' ? (
            <DeliveriesView />
          ) : currentView === 'dashboard' ? (
            <DashboardView
              onGoToReader={() => handleSelectView('unread', { title: 'Articles non lus' })}
              onGoToDeliveries={() => handleSelectView('deliveries', { title: 'Livraisons' })}
            />
          ) : currentView === 'settings' ? (
            <SettingsView />
          ) : (
            <ReaderView
              view={currentView.startsWith('feed-') || currentView.startsWith('col-') ? '' : currentView}
              feedId={viewMeta.feedId}
              categoryId={viewMeta.categoryId}
              collectionId={viewMeta.collectionId}
              title={viewTitle}
            />
          )}
        </main>
      </div>

      {/* Drawer & Modal */}
      <KindleCartDrawer />
      <LoginModal />
    </div>
  );
};

export const App: React.FC = () => {
  return (
    <AuthProvider>
      <CartProvider>
        <MainApp />
        <Toaster position="bottom-right" richColors theme="system" />
      </CartProvider>
    </AuthProvider>
  );
};
