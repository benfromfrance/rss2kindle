import React, { useState } from 'react';
import type { Category, Counts, Feed, Collection } from '../types';
import {
  Inbox,
  Layers,
  Star,
  Package,
  Calendar,
  Rss,
  Folder,
  Send,
  BarChart2,
  Settings as SettingsIcon,
  RefreshCw,
  Moon,
  Sun,
  LogOut,
  ChevronRight,
  ChevronDown,
  Plus,
  BookOpen,
} from 'lucide-react';

interface SidebarProps {
  currentView: string;
  onSelectView: (view: string, meta?: { feedId?: number; categoryId?: number; collectionId?: number; title?: string }) => void;
  counts: Counts | null;
  categories: Category[];
  feeds: Feed[];
  collections: Collection[];
  onRefreshAll: () => void;
  isRefreshing: boolean;
  onLogout: () => void;
  isDark: boolean;
  onToggleDark: () => void;
}

export const Sidebar: React.FC<SidebarProps> = ({
  currentView,
  onSelectView,
  counts,
  categories,
  feeds,
  collections,
  onRefreshAll,
  isRefreshing,
  onLogout,
  isDark,
  onToggleDark,
}) => {
  const [collapsedCats, setCollapsedCats] = useState<Record<number, boolean>>({});

  const toggleCategory = (catId: number) => {
    setCollapsedCats((prev) => ({ ...prev, [catId]: !prev[catId] }));
  };

  // Regroupement des flux par catégorie
  const feedsByCat: Record<number, Feed[]> = {};
  const uncategorizedFeeds: Feed[] = [];

  feeds.forEach((f) => {
    if (f.category_id) {
      if (!feedsByCat[f.category_id]) feedsByCat[f.category_id] = [];
      feedsByCat[f.category_id].push(f);
    } else {
      uncategorizedFeeds.push(f);
    }
  });

  return (
    <aside className="w-64 h-full bg-stone-100 dark:bg-stone-900 border-r border-stone-200 dark:border-stone-800 flex flex-col shrink-0 select-none">
      {/* Brand / Logo */}
      <div className="p-4 border-b border-stone-200 dark:border-stone-800 flex items-center justify-between">
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-lg bg-amber-600 flex items-center justify-center text-white shadow-xs">
            <BookOpen className="w-5 h-5" />
          </div>
          <div>
            <h1 className="font-bold text-sm tracking-tight text-stone-900 dark:text-stone-100">
              RSS to Kindle
            </h1>
            <span className="text-[10px] text-stone-400 font-mono block -mt-0.5">
              E-INK DIGEST
            </span>
          </div>
        </div>

        {/* Bouton rafraîchir tous les flux */}
        <button
          onClick={onRefreshAll}
          disabled={isRefreshing}
          className="p-1.5 rounded-lg text-stone-500 hover:text-stone-900 dark:hover:text-stone-100 hover:bg-stone-200 dark:hover:bg-stone-800 transition-colors disabled:opacity-40"
          title="Rafraîchir tous les flux RSS"
        >
          <RefreshCw className={`w-4 h-4 ${isRefreshing ? 'animate-spin text-amber-600' : ''}`} />
        </button>
      </div>

      {/* Navigation scrollable */}
      <div className="flex-1 overflow-y-auto p-3 space-y-6 text-xs">
        {/* Vues principales */}
        <div className="space-y-0.5">
          <NavItem
            icon={<Inbox className="w-4 h-4" />}
            label="Non lus"
            count={counts?.unread}
            active={currentView === 'unread'}
            onClick={() => onSelectView('unread', { title: 'Articles non lus' })}
          />
          <NavItem
            icon={<Layers className="w-4 h-4" />}
            label="Tous les articles"
            count={counts?.total}
            active={currentView === 'all'}
            onClick={() => onSelectView('all', { title: 'Tous les articles' })}
          />
          <NavItem
            icon={<Calendar className="w-4 h-4" />}
            label="Aujourd'hui"
            count={counts?.today}
            active={currentView === 'today'}
            onClick={() => onSelectView('today', { title: "Articles d'aujourd'hui" })}
          />
          <NavItem
            icon={<Star className="w-4 h-4" />}
            label="Favoris"
            count={counts?.starred}
            active={currentView === 'starred'}
            onClick={() => onSelectView('starred', { title: 'Articles favoris' })}
          />
          <NavItem
            icon={<Package className="w-4 h-4" />}
            label="Pas encore envoyés"
            count={counts?.unsent}
            active={currentView === 'unsent'}
            onClick={() => onSelectView('unsent', { title: 'Pas encore envoyés au Kindle' })}
          />
        </div>

        {/* Catégories & Flux */}
        <div className="space-y-2">
          <div className="flex items-center justify-between px-2 text-[11px] font-semibold tracking-wider text-stone-400 uppercase">
            <span>Abonnements</span>
            <button
              onClick={() => onSelectView('feeds', { title: 'Gestion des flux' })}
              className="hover:text-stone-800 dark:hover:text-stone-200"
              title="Ajouter un flux"
            >
              <Plus className="w-3.5 h-3.5" />
            </button>
          </div>

          <div className="space-y-1">
            {categories.map((cat) => {
              const catFeeds = feedsByCat[cat.id] || [];
              const isCollapsed = collapsedCats[cat.id];
              const unreadTotal = counts?.by_category?.[cat.id] || 0;

              return (
                <div key={cat.id} className="space-y-0.5">
                  <div
                    onClick={() => toggleCategory(cat.id)}
                    className="flex items-center justify-between px-2 py-1.5 rounded-lg text-stone-600 dark:text-stone-400 hover:bg-stone-200/60 dark:hover:bg-stone-800/60 cursor-pointer font-medium"
                  >
                    <div className="flex items-center gap-1.5 truncate">
                      {isCollapsed ? (
                        <ChevronRight className="w-3.5 h-3.5 text-stone-400" />
                      ) : (
                        <ChevronDown className="w-3.5 h-3.5 text-stone-400" />
                      )}
                      <span className="truncate">{cat.name}</span>
                    </div>
                    {unreadTotal > 0 && (
                      <span className="text-[10px] font-semibold text-stone-400">
                        {unreadTotal}
                      </span>
                    )}
                  </div>

                  {/* Flux de la catégorie */}
                  {!isCollapsed && (
                    <div className="pl-4 space-y-0.5">
                      {catFeeds.map((feed) => {
                        const feedUnread = counts?.by_feed?.[feed.id] || 0;
                        const isCurrent = currentView === `feed-${feed.id}`;
                        return (
                          <div
                            key={feed.id}
                            onClick={() =>
                              onSelectView(`feed-${feed.id}`, {
                                feedId: feed.id,
                                title: feed.title,
                              })
                            }
                            className={`flex items-center justify-between px-2 py-1 rounded-lg cursor-pointer transition-colors ${
                              isCurrent
                                ? 'bg-amber-100 dark:bg-amber-950/60 text-amber-900 dark:text-amber-200 font-medium'
                                : 'text-stone-600 dark:text-stone-400 hover:bg-stone-200/50 dark:hover:bg-stone-800/50'
                            }`}
                          >
                            <span className="truncate">{feed.title}</span>
                            {feedUnread > 0 && (
                              <span className="text-[10px] bg-stone-200 dark:bg-stone-800 px-1.5 py-0.2 rounded font-mono">
                                {feedUnread}
                              </span>
                            )}
                          </div>
                        );
                      })}
                    </div>
                  )}
                </div>
              );
            })}

            {/* Flux sans catégorie */}
            {uncategorizedFeeds.map((feed) => {
              const feedUnread = counts?.by_feed?.[feed.id] || 0;
              const isCurrent = currentView === `feed-${feed.id}`;
              return (
                <div
                  key={feed.id}
                  onClick={() =>
                    onSelectView(`feed-${feed.id}`, {
                      feedId: feed.id,
                      title: feed.title,
                    })
                  }
                  className={`flex items-center justify-between px-2 py-1 rounded-lg cursor-pointer transition-colors ${
                    isCurrent
                      ? 'bg-amber-100 dark:bg-amber-950/60 text-amber-900 dark:text-amber-200 font-medium'
                      : 'text-stone-600 dark:text-stone-400 hover:bg-stone-200/50 dark:hover:bg-stone-800/50'
                  }`}
                >
                  <span className="truncate">{feed.title}</span>
                  {feedUnread > 0 && (
                    <span className="text-[10px] bg-stone-200 dark:bg-stone-800 px-1.5 py-0.2 rounded font-mono">
                      {feedUnread}
                    </span>
                  )}
                </div>
              );
            })}
          </div>
        </div>

        {/* Collections */}
        {collections.length > 0 && (
          <div className="space-y-1">
            <div className="px-2 text-[11px] font-semibold tracking-wider text-stone-400 uppercase">
              Collections
            </div>
            {collections.map((col) => (
              <div
                key={col.id}
                onClick={() =>
                  onSelectView(`col-${col.id}`, {
                    collectionId: col.id,
                    title: `Collection : ${col.name}`,
                  })
                }
                className={`flex items-center justify-between px-2 py-1 rounded-lg cursor-pointer ${
                  currentView === `col-${col.id}`
                    ? 'bg-amber-100 dark:bg-amber-950/60 text-amber-900 dark:text-amber-200 font-medium'
                    : 'text-stone-600 dark:text-stone-400 hover:bg-stone-200/50 dark:hover:bg-stone-800/50'
                }`}
              >
                <div className="flex items-center gap-1.5 truncate">
                  <Folder className="w-3.5 h-3.5 text-stone-400" />
                  <span className="truncate">{col.name}</span>
                </div>
                <span className="text-[10px] text-stone-400">{col.article_count}</span>
              </div>
            ))}
          </div>
        )}

        {/* Pages d'administration */}
        <div className="space-y-0.5 pt-2 border-t border-stone-200 dark:border-stone-800">
          <NavItem
            icon={<Rss className="w-4 h-4" />}
            label="Gestion des flux & OPML"
            active={currentView === 'feeds'}
            onClick={() => onSelectView('feeds', { title: 'Gestion des flux' })}
          />
          <NavItem
            icon={<Send className="w-4 h-4" />}
            label="Livraisons Kindle"
            active={currentView === 'deliveries'}
            onClick={() => onSelectView('deliveries', { title: 'Livraisons Kindle' })}
          />
          <NavItem
            icon={<BarChart2 className="w-4 h-4" />}
            label="Tableau de bord"
            active={currentView === 'dashboard'}
            onClick={() => onSelectView('dashboard', { title: 'Tableau de bord' })}
          />
          <NavItem
            icon={<SettingsIcon className="w-4 h-4" />}
            label="Paramètres"
            active={currentView === 'settings'}
            onClick={() => onSelectView('settings', { title: 'Paramètres' })}
          />
        </div>
      </div>

      {/* Pied de sidebar : Dark mode & déconnexion */}
      <div className="p-3 border-t border-stone-200 dark:border-stone-800 flex items-center justify-between text-xs text-stone-500">
        <button
          onClick={onToggleDark}
          className="p-1.5 rounded-lg hover:bg-stone-200 dark:hover:bg-stone-800 text-stone-600 dark:text-stone-300 transition-colors flex items-center gap-1.5"
          title="Basculer le thème clair / sombre"
        >
          {isDark ? <Sun className="w-4 h-4" /> : <Moon className="w-4 h-4" />}
          <span>{isDark ? 'Clair' : 'Sombre'}</span>
        </button>

        <button
          onClick={onLogout}
          className="p-1.5 rounded-lg hover:bg-stone-200 dark:hover:bg-stone-800 text-stone-500 hover:text-red-600 dark:hover:text-red-400 transition-colors"
          title="Se déconnecter"
        >
          <LogOut className="w-4 h-4" />
        </button>
      </div>
    </aside>
  );
};

const NavItem: React.FC<{
  icon: React.ReactNode;
  label: string;
  count?: number;
  active?: boolean;
  onClick: () => void;
}> = ({ icon, label, count, active, onClick }) => (
  <button
    onClick={onClick}
    className={`w-full flex items-center justify-between px-2.5 py-1.5 rounded-lg transition-colors ${
      active
        ? 'bg-amber-100 dark:bg-amber-950/60 text-amber-900 dark:text-amber-200 font-semibold'
        : 'text-stone-700 dark:text-stone-300 hover:bg-stone-200/60 dark:hover:bg-stone-800/60'
    }`}
  >
    <div className="flex items-center gap-2 truncate">
      <span className={active ? 'text-amber-600 dark:text-amber-400' : 'text-stone-500'}>
        {icon}
      </span>
      <span className="truncate">{label}</span>
    </div>
    {count !== undefined && count > 0 && (
      <span
        className={`text-[11px] font-mono px-1.5 py-0.2 rounded-full ${
          active
            ? 'bg-amber-200/80 dark:bg-amber-900 text-amber-900 dark:text-amber-100 font-bold'
            : 'bg-stone-200 dark:bg-stone-800 text-stone-600 dark:text-stone-400'
        }`}
      >
        {count}
      </span>
    )}
  </button>
);
