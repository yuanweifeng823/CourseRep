const { withAndroidManifest } = require('@expo/config-plugins');

module.exports = ({ config }) => withAndroidManifest({
  ...config,
  android: { ...config.android },
}, (config) => {
  const app = config.modResults.manifest.application?.[0];
  if (app) {
    // Local phone testing uses HTTP on the LAN; production builds require HTTPS.
    app.$['android:usesCleartextTraffic'] = process.env.APP_ENV === 'production' ? 'false' : 'true';
  }
  return config;
});
