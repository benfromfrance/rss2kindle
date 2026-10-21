import React, { useEffect, useState } from 'react';
import type { Settings } from '../types';
import { api } from '../lib/api';
import {
  Save,
  Mail,
  Clock,
  Image as ImageIcon,
  ShieldCheck,
  Send,
  HelpCircle,
} from 'lucide-react';
import { toast } from 'sonner';

export const SettingsView: React.FC = () => {
  const [settings, setSettings] = useState<Settings | null>(null);
  const [hasPass, setHasPass] = useState(false);
  const [newPassword, setNewPassword] = useState('');
  const [isSaving, setIsSaving] = useState(false);
  const [isTestingEmail, setIsTestingEmail] = useState(false);

  useEffect(() => {
    api.getSettings().then((res) => {
      setSettings(res);
      setHasPass(res.has_smtp_password);
    });
  }, []);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!settings) return;
    setIsSaving(true);
    try {
      const payload: any = { ...settings };
      if (newPassword) {
        payload.smtp_password = newPassword;
      }
      const res = await api.updateSettings(payload);
      setSettings(res.settings);
      setHasPass(res.has_smtp_password);
      setNewPassword('');
      toast.success('Paramètres enregistrés avec succès !');
    } catch (err: any) {
      toast.error('Erreur enregistrement: ' + err.message);
    } finally {
      setIsSaving(false);
    }
  };

  const handleTestEmail = async () => {
    if (!settings) return;
    setIsTestingEmail(true);
    try {
      const res = await api.testEmail({
        smtp_host: settings.smtp_host,
        smtp_port: settings.smtp_port,
        smtp_user: settings.smtp_user,
        smtp_password: newPassword || undefined,
        smtp_security: settings.smtp_security,
        sender_email: settings.sender_email,
        sender_name: settings.sender_name,
        kindle_email: settings.kindle_email,
        recipient: settings.sender_email || settings.kindle_email,
      });
      toast.success(res.message);
    } catch (err: any) {
      toast.error(err.message || 'Échec du test SMTP');
    } finally {
      setIsTestingEmail(false);
    }
  };

  if (!settings) {
    return <div className="p-8 text-center text-stone-500">Chargement des paramètres...</div>;
  }

  const daysLabels = ['Dim', 'Lun', 'Mar', 'Mer', 'Jeu', 'Ven', 'Sam'];

  return (
    <div className="flex-1 h-full overflow-y-auto p-6 md:p-8 bg-stone-50 dark:bg-stone-950">
      <form onSubmit={handleSave} className="max-w-4xl mx-auto space-y-8 pb-12">
        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-6 border-b border-stone-200 dark:border-stone-800">
          <div>
            <h1 className="text-2xl font-bold text-stone-900 dark:text-stone-100">
              Paramètres de l'application
            </h1>
            <p className="text-sm text-stone-500">
              Configurez votre compte Kindle, les identifiants SMTP et la planification quotidienne.
            </p>
          </div>

          <button
            type="submit"
            disabled={isSaving}
            className="px-5 py-2.5 text-xs font-semibold bg-amber-600 hover:bg-amber-700 text-white rounded-lg flex items-center gap-2 shadow-xs transition-colors disabled:opacity-50"
          >
            <Save className="w-4 h-4" />
            {isSaving ? 'Enregistrement...' : 'Enregistrer les modifications'}
          </button>
        </div>

        {/* 1. Configuration Kindle & Envoi Email */}
        <section className="bg-white dark:bg-stone-900 rounded-xl border border-stone-200 dark:border-stone-800 p-6 space-y-5 shadow-xs">
          <div className="flex items-center gap-2 text-stone-900 dark:text-stone-100 font-semibold text-base border-b border-stone-100 dark:border-stone-800 pb-3">
            <Mail className="w-5 h-5 text-amber-600" />
            <h2>Kindle & Serveur SMTP</h2>
          </div>

          {/* Alert Amazon Whitelist */}
          <div className="p-3.5 bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-900/60 rounded-xl text-xs text-amber-900 dark:text-amber-200 space-y-1">
            <div className="font-semibold flex items-center gap-1.5">
              <ShieldCheck className="w-4 h-4 text-amber-600" />
              Important pour la réception Kindle :
            </div>
            <p>
              Pour qu'Amazon autorise l'arrivée des livres, vous <strong>devez</strong> ajouter votre{' '}
              <em>Adresse email d'expédition</em> dans la liste des expéditeurs approuvés de documents personnels de votre compte Amazon (rubrique <em>Gérer votre contenu et vos appareils → Préférences → Paramètres des documents personnels</em>).
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
            <div>
              <label className="block font-semibold text-stone-700 dark:text-stone-300 mb-1">
                Adresse email Kindle (@kindle.com)
              </label>
              <input
                type="email"
                required
                value={settings.kindle_email}
                onChange={(e) => setSettings({ ...settings, kindle_email: e.target.value })}
                placeholder="votre-liseuse@kindle.com"
                className="w-full px-3 py-2 text-sm rounded-lg border border-stone-300 dark:border-stone-700 bg-stone-50 dark:bg-stone-800 text-stone-900 dark:text-stone-100"
              />
            </div>

            <div>
              <label className="block font-semibold text-stone-700 dark:text-stone-300 mb-1">
                Adresse email d'expédition approuvée
              </label>
              <input
                type="email"
                required
                value={settings.sender_email}
                onChange={(e) => setSettings({ ...settings, sender_email: e.target.value })}
                placeholder="votre-email@gmail.com"
                className="w-full px-3 py-2 text-sm rounded-lg border border-stone-300 dark:border-stone-700 bg-stone-50 dark:bg-stone-800 text-stone-900 dark:text-stone-100"
              />
            </div>

            <div>
              <label className="block font-semibold text-stone-700 dark:text-stone-300 mb-1">
                Hôte SMTP
              </label>
              <input
                type="text"
                required
                value={settings.smtp_host}
                onChange={(e) => setSettings({ ...settings, smtp_host: e.target.value })}
                placeholder="smtp.gmail.com ou mail.votre-domaine.com"
                className="w-full px-3 py-2 text-sm rounded-lg border border-stone-300 dark:border-stone-700 bg-stone-50 dark:bg-stone-800 text-stone-900 dark:text-stone-100"
              />
            </div>

            <div className="grid grid-cols-2 gap-2">
              <div>
                <label className="block font-semibold text-stone-700 dark:text-stone-300 mb-1">
                  Port SMTP
                </label>
                <input
                  type="number"
                  required
                  value={settings.smtp_port}
                  onChange={(e) => setSettings({ ...settings, smtp_port: Number(e.target.value) })}
                  className="w-full px-3 py-2 text-sm rounded-lg border border-stone-300 dark:border-stone-700 bg-stone-50 dark:bg-stone-800 text-stone-900 dark:text-stone-100"
                />
              </div>
              <div>
                <label className="block font-semibold text-stone-700 dark:text-stone-300 mb-1">
                  Sécurité
                </label>
                <select
                  value={settings.smtp_security}
                  onChange={(e) =>
                    setSettings({ ...settings, smtp_security: e.target.value as any })
                  }
                  className="w-full px-3 py-2 text-sm rounded-lg border border-stone-300 dark:border-stone-700 bg-stone-50 dark:bg-stone-800 text-stone-900 dark:text-stone-100"
                >
                  <option value="starttls">STARTTLS (587)</option>
                  <option value="tls">SSL/TLS (465)</option>
                  <option value="none">Aucune (25)</option>
                </select>
              </div>
            </div>

            <div>
              <label className="block font-semibold text-stone-700 dark:text-stone-300 mb-1">
                Identifiant / Utilisateur SMTP
              </label>
              <input
                type="text"
                value={settings.smtp_user}
                onChange={(e) => setSettings({ ...settings, smtp_user: e.target.value })}
                className="w-full px-3 py-2 text-sm rounded-lg border border-stone-300 dark:border-stone-700 bg-stone-50 dark:bg-stone-800 text-stone-900 dark:text-stone-100"
              />
            </div>

            <div>
              <label className="block font-semibold text-stone-700 dark:text-stone-300 mb-1">
                Mot de passe SMTP {hasPass && '(déjà configuré et chiffré)'}
              </label>
              <input
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                placeholder={hasPass ? 'Laisser vide pour conserver' : 'Saisir mot de passe'}
                className="w-full px-3 py-2 text-sm rounded-lg border border-stone-300 dark:border-stone-700 bg-stone-50 dark:bg-stone-800 text-stone-900 dark:text-stone-100"
              />
            </div>
          </div>

          <div className="pt-2">
            <button
              type="button"
              disabled={isTestingEmail || !settings.smtp_host}
              onClick={handleTestEmail}
              className="px-3.5 py-2 text-xs font-medium bg-stone-100 dark:bg-stone-800 hover:bg-stone-200 dark:hover:bg-stone-700 text-stone-800 dark:text-stone-200 rounded-lg flex items-center gap-1.5 transition-colors disabled:opacity-50"
            >
              <Send className="w-3.5 h-3.5" />
              {isTestingEmail ? 'Test en cours...' : 'Envoyer un email de test SMTP'}
            </button>
          </div>
        </section>

        {/* 2. Planification quotidienne (Digest à heure fixe) */}
        <section className="bg-white dark:bg-stone-900 rounded-xl border border-stone-200 dark:border-stone-800 p-6 space-y-5 shadow-xs">
          <div className="flex items-center gap-2 text-stone-900 dark:text-stone-100 font-semibold text-base border-b border-stone-100 dark:border-stone-800 pb-3">
            <Clock className="w-5 h-5 text-amber-600" />
            <h2>Digest quotidien automatique</h2>
          </div>

          <div className="space-y-4 text-xs">
            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={settings.digest_enabled}
                onChange={(e) => setSettings({ ...settings, digest_enabled: e.target.checked })}
                className="w-4 h-4 rounded text-amber-600"
              />
              <span className="font-semibold text-sm text-stone-900 dark:text-stone-100">
                Activer l'envoi automatique chaque jour
              </span>
            </label>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 pt-2">
              <div>
                <label className="block font-semibold text-stone-700 dark:text-stone-300 mb-1">
                  Heure de livraison
                </label>
                <input
                  type="text"
                  required
                  value={settings.digest_time}
                  onChange={(e) => setSettings({ ...settings, digest_time: e.target.value })}
                  placeholder="01:00"
                  className="w-full px-3 py-2 text-sm rounded-lg border border-stone-300 dark:border-stone-700 bg-stone-50 dark:bg-stone-800 text-stone-900 dark:text-stone-100 font-mono"
                />
                <span className="text-[11px] text-stone-400 mt-0.5 block">Format HH:MM (ex: 01:00)</span>
              </div>

              <div>
                <label className="block font-semibold text-stone-700 dark:text-stone-300 mb-1">
                  Fuseau horaire
                </label>
                <input
                  type="text"
                  required
                  value={settings.timezone}
                  onChange={(e) => setSettings({ ...settings, timezone: e.target.value })}
                  placeholder="Europe/Paris"
                  className="w-full px-3 py-2 text-sm rounded-lg border border-stone-300 dark:border-stone-700 bg-stone-50 dark:bg-stone-800 text-stone-900 dark:text-stone-100"
                />
              </div>

              <div>
                <label className="block font-semibold text-stone-700 dark:text-stone-300 mb-1">
                  Fenêtre de fraîcheur (heures)
                </label>
                <input
                  type="number"
                  min="1"
                  max="720"
                  value={settings.digest_window_hours}
                  onChange={(e) =>
                    setSettings({ ...settings, digest_window_hours: Number(e.target.value) })
                  }
                  className="w-full px-3 py-2 text-sm rounded-lg border border-stone-300 dark:border-stone-700 bg-stone-50 dark:bg-stone-800 text-stone-900 dark:text-stone-100"
                />
              </div>
            </div>

            {/* Jours de livraison */}
            <div>
              <label className="block font-semibold text-stone-700 dark:text-stone-300 mb-2">
                Jours de livraison
              </label>
              <div className="flex gap-2">
                {daysLabels.map((day, idx) => {
                  const selected = settings.digest_days.includes(idx);
                  return (
                    <button
                      type="button"
                      key={day}
                      onClick={() => {
                        const newDays = selected
                          ? settings.digest_days.filter((d) => d !== idx)
                          : [...settings.digest_days, idx].sort();
                        setSettings({ ...settings, digest_days: newDays });
                      }}
                      className={`px-3 py-1.5 rounded-lg font-medium transition-colors ${
                        selected
                          ? 'bg-amber-600 text-white'
                          : 'bg-stone-100 dark:bg-stone-800 text-stone-600 dark:text-stone-400 hover:bg-stone-200'
                      }`}
                    >
                      {day}
                    </button>
                  );
                })}
              </div>
            </div>

            <div className="space-y-2 pt-2">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={settings.digest_exclude_read}
                  onChange={(e) =>
                    setSettings({ ...settings, digest_exclude_read: e.target.checked })
                  }
                  className="rounded text-amber-600"
                />
                <span>Ne pas inclure les articles déjà marqués comme lus dans l'application</span>
              </label>

              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={settings.digest_skip_empty}
                  onChange={(e) =>
                    setSettings({ ...settings, digest_skip_empty: e.target.checked })
                  }
                  className="rounded text-amber-600"
                />
                <span>Ne rien envoyer si aucun nouvel article n'a été publié</span>
              </label>
            </div>
          </div>
        </section>

        {/* 3. Optimisation E-ink & Images */}
        <section className="bg-white dark:bg-stone-900 rounded-xl border border-stone-200 dark:border-stone-800 p-6 space-y-4 shadow-xs">
          <div className="flex items-center gap-2 text-stone-900 dark:text-stone-100 font-semibold text-base border-b border-stone-100 dark:border-stone-800 pb-3">
            <ImageIcon className="w-5 h-5 text-amber-600" />
            <h2>Optimisations pour liseuse E-ink</h2>
          </div>

          <div className="space-y-3 text-xs">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={settings.images_enabled}
                onChange={(e) => setSettings({ ...settings, images_enabled: e.target.checked })}
                className="rounded text-amber-600"
              />
              <span>Télécharger et inclure les images dans les livres</span>
            </label>

            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={settings.images_grayscale}
                onChange={(e) => setSettings({ ...settings, images_grayscale: e.target.checked })}
                className="rounded text-amber-600"
              />
              <span>Convertir les images en niveaux de gris (réduit le poids du livre)</span>
            </label>

            <div className="w-48 pt-2">
              <label className="block font-semibold text-stone-700 dark:text-stone-300 mb-1">
                Largeur max image (pixels)
              </label>
              <input
                type="number"
                value={settings.image_max_width}
                onChange={(e) =>
                  setSettings({ ...settings, image_max_width: Number(e.target.value) })
                }
                className="w-full px-3 py-2 text-sm rounded-lg border border-stone-300 dark:border-stone-700 bg-stone-50 dark:bg-stone-800 text-stone-900 dark:text-stone-100"
              />
            </div>
          </div>
        </section>
      </form>
    </div>
  );
};
