import { useState } from 'react';
import { ActivityIndicator, Alert, StyleSheet, Text, TextInput, TouchableOpacity, View } from 'react-native';
import { apiBaseUrl, resetApiBase, setApiBase } from '../../api';
import { colors } from './theme';

// 允许在应用内切换后端地址：局域网 IP、公网隧道或正式域名都可以，不必重新打包。
export function ServerField({ title = '服务器地址' }: { title?: string }) {
  const [open, setOpen] = useState(false);
  const [value, setValue] = useState(apiBaseUrl);
  const [busy, setBusy] = useState(false);

  const save = async () => {
    const saved = await setApiBase(value);
    if (!saved) {
      Alert.alert('地址无效', '请输入形如 http://192.168.1.2:18080 或 https://xxx.trycloudflare.com 的地址');
      return;
    }
    setBusy(true);
    try {
      const controller = new AbortController();
      const timer = setTimeout(() => controller.abort(), 8000);
      const response = await fetch(`${apiBaseUrl}/healthz`, { signal: controller.signal }).catch(() => null);
      clearTimeout(timer);
      Alert.alert(
        response?.ok ? '已保存并连接成功' : '已保存，但服务器未响应',
        response?.ok ? `当前服务器：${apiBaseUrl}` : `无法访问 ${apiBaseUrl}/healthz，请检查地址、网络或隧道是否在运行。`,
      );
      setValue(apiBaseUrl);
      setOpen(false);
    } finally {
      setBusy(false);
    }
  };

  const restore = async () => {
    await resetApiBase();
    setValue(apiBaseUrl);
    Alert.alert('已恢复默认', apiBaseUrl);
  };

  return (
    <View style={styles.wrap}>
      <TouchableOpacity onPress={() => { setValue(apiBaseUrl); setOpen((prev) => !prev); }} style={styles.row}>
        <View style={{ flex: 1 }}>
          <Text style={styles.label}>{title}</Text>
          <Text style={styles.current} numberOfLines={1}>{apiBaseUrl}</Text>
        </View>
        <Text style={styles.action}>{open ? '收起' : '更改'}</Text>
      </TouchableOpacity>
      {open && (
        <View style={styles.editor}>
          <TextInput
            autoCapitalize="none"
            autoCorrect={false}
            value={value}
            onChangeText={setValue}
            placeholder="http://192.168.1.2:18080"
            placeholderTextColor={colors.muted}
            style={styles.input}
          />
          <Text style={styles.hint}>不在同一 Wi-Fi 时，填公网隧道或正式域名，例如 https://xxx.trycloudflare.com</Text>
          <View style={styles.buttons}>
            <TouchableOpacity disabled={busy} onPress={() => void save()} style={[styles.primary, busy && { opacity: 0.6 }]}>
              {busy ? <ActivityIndicator color="#fff" size="small" /> : <Text style={styles.primaryText}>保存并测试</Text>}
            </TouchableOpacity>
            <TouchableOpacity disabled={busy} onPress={() => void restore()} style={styles.secondary}>
              <Text style={styles.secondaryText}>恢复默认</Text>
            </TouchableOpacity>
          </View>
        </View>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { backgroundColor: colors.card, borderRadius: 14, borderWidth: 1, borderColor: colors.line, padding: 14, marginTop: 12 },
  row: { flexDirection: 'row', alignItems: 'center' },
  label: { color: colors.muted, fontSize: 11, fontWeight: '700' },
  current: { color: colors.ink, fontSize: 13, marginTop: 2 },
  action: { color: colors.green, fontSize: 12, fontWeight: '800', marginLeft: 12 },
  editor: { marginTop: 12 },
  input: { borderWidth: 1, borderColor: colors.line, borderRadius: 10, paddingHorizontal: 12, paddingVertical: 10, color: colors.ink, backgroundColor: '#fff' },
  hint: { color: colors.muted, fontSize: 11, lineHeight: 16, marginTop: 8 },
  buttons: { flexDirection: 'row', gap: 10, marginTop: 12 },
  primary: { backgroundColor: colors.green, borderRadius: 10, paddingHorizontal: 16, paddingVertical: 10, alignItems: 'center', justifyContent: 'center', minWidth: 110 },
  primaryText: { color: '#fff', fontSize: 13, fontWeight: '800' },
  secondary: { borderWidth: 1, borderColor: colors.line, borderRadius: 10, paddingHorizontal: 16, paddingVertical: 10 },
  secondaryText: { color: colors.ink, fontSize: 13, fontWeight: '700' },
});
