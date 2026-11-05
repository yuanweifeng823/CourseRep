import { StatusBar } from 'expo-status-bar';
import { useCallback, useEffect, useState } from 'react';
import * as SecureStore from 'expo-secure-store';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { CourseDetail } from './CourseDetail';
import { AdminScreen, NotificationsScreen, ProfileEditScreen } from './UserScreens';
import { apiFetch, initApiBase } from './api';
import type { Course, FeedPost, LoginRequest, LoginResponse, RegisterRequest } from './src/api/types';
import { CourseCatalog } from './src/screens/CourseCatalog';
import { ComposePostScreen } from './src/screens/ComposePostScreen';
import { HomeScreen } from './src/screens/HomeScreen';
import { MeScreen } from './src/screens/MeScreen';
import { MyListScreen, type EditTarget, type MyListKind } from './src/screens/MyListScreen';
import { PostDetailScreen } from './src/screens/PostDetailScreen';
import { ShareFeedScreen } from './src/screens/ShareFeedScreen';
import { ServerField } from './src/ui/ServerField';
import { colors } from './src/ui/theme';
import {
  ActivityIndicator,
  BackHandler,
  SafeAreaView,
  ScrollView,
  StatusBar as NativeStatusBar,
  StyleSheet,
  Text,
  TextInput,
  TouchableOpacity,
  View,
} from 'react-native';

type Tab = 'home' | 'courses' | 'share' | 'notifications' | 'me';

type Route =
  | { kind: 'course'; course: Course }
  | { kind: 'post'; postId: number; course: Course; seed?: Partial<FeedPost> }
  | { kind: 'list'; list: MyListKind }
  | { kind: 'admin' }
  | { kind: 'compose' }
  | { kind: 'edit'; edit: EditTarget };

const tabs: { key: Tab; icon: string; label: string }[] = [
  { key: 'home', icon: '⌂', label: '首页' },
  { key: 'courses', icon: '▤', label: '课程' },
  { key: 'share', icon: '＋', label: '分享' },
  { key: 'notifications', icon: '♧', label: '通知' },
  { key: 'me', icon: '◉', label: '我的' },
];

const apiBaseUrl = (process.env.EXPO_PUBLIC_API_URL || 'http://localhost:8080').replace(/\/$/, '');
const queryClient = new QueryClient({ defaultOptions: { queries: { staleTime: 15_000, retry: 1 } } });

function AuthScreen({ onLogin }: { onLogin: (access: string, refresh: string) => Promise<void> }) {
  const [mode, setMode] = useState<'login' | 'register' | 'verify'>('login');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [nickname, setNickname] = useState('');
  const [code, setCode] = useState('');
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState('');
  const submit = async () => {
    setBusy(true);
    setMessage('');
    try {
      const path = mode === 'verify' ? 'verify' : mode;
      const body: LoginRequest | RegisterRequest | { email: string; code: string } =
        mode === 'login' ? { email, password } : mode === 'register' ? { email, password, nickname } : { email, code };
      const response = await fetch(`${apiBaseUrl}/api/v1/auth/${path}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      const payload = (await response.json()) as LoginResponse & { error?: string; message?: string };
      if (!response.ok) throw new Error(payload.error ?? '请求失败，请稍后再试');
      if (mode === 'login' && payload.access_token && payload.refresh_token) await onLogin(payload.access_token, payload.refresh_token);
      else if (mode === 'register') {
        setMode('verify');
        setMessage(payload.message ?? '验证码已发送');
      } else {
        setMode('login');
        setMessage('邮箱验证完成，请登录');
      }
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '暂时无法连接服务');
    } finally {
      setBusy(false);
    }
  };
  const title = mode === 'login' ? '登录 CourseRep' : mode === 'register' ? '创建校内账号' : '验证学生邮箱';
  return (
    <SafeAreaView style={styles.safe}>
      <ScrollView contentContainerStyle={styles.authContent} keyboardShouldPersistTaps="handled">
        <View style={styles.mark}>
          <Text style={styles.markText}>C</Text>
        </View>
        <Text style={styles.authTitle}>{title}</Text>
        <Text style={styles.authIntro}>使用学校允许的学生邮箱域名注册</Text>
        <TextInput
          autoCapitalize="none"
          keyboardType="email-address"
          value={email}
          onChangeText={setEmail}
          placeholder="学生邮箱"
          style={styles.authInput}
        />
        {mode !== 'verify' && (
          <TextInput value={password} onChangeText={setPassword} secureTextEntry placeholder="密码（至少 10 位）" style={styles.authInput} />
        )}
        {mode === 'register' && <TextInput value={nickname} onChangeText={setNickname} placeholder="昵称" style={styles.authInput} />}
        {mode === 'verify' && (
          <TextInput value={code} onChangeText={setCode} keyboardType="number-pad" maxLength={6} placeholder="邮箱验证码" style={styles.authInput} />
        )}
        {!!message && <Text style={styles.authMessage}>{message}</Text>}
        <TouchableOpacity disabled={busy} onPress={() => void submit()} style={styles.authButton}>
          <Text style={styles.authButtonText}>
            {busy ? '请稍候…' : mode === 'login' ? '登录' : mode === 'register' ? '发送验证码' : '验证邮箱'}
          </Text>
        </TouchableOpacity>
        <TouchableOpacity
          onPress={() => {
            setMessage('');
            setMode(mode === 'login' ? 'register' : 'login');
          }}
          style={{ padding: 16 }}
        >
          <Text style={styles.authSwitch}>{mode === 'login' ? '还没有账号？注册' : '返回登录'}</Text>
        </TouchableOpacity>
        {mode === 'verify' && (
          <TouchableOpacity onPress={() => setMode('register')}>
            <Text style={styles.authSwitch}>修改注册信息</Text>
          </TouchableOpacity>
        )}
        <ServerField />
      </ScrollView>
    </SafeAreaView>
  );
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AppShell />
    </QueryClientProvider>
  );
}

function AppShell() {
  const [authLoaded, setAuthLoaded] = useState(false);
  const [accessToken, setAccessToken] = useState('');
  const [tab, setTab] = useState<Tab>('home');
  const [routes, setRoutes] = useState<Route[]>([]);

  useEffect(() => {
    void (async () => {
      await initApiBase();
      let token = await SecureStore.getItemAsync('access_token');
      const refresh = await SecureStore.getItemAsync('refresh_token');
      if (token) {
        const response = await fetch(`${apiBaseUrl}/api/v1/auth/me`, { headers: { Authorization: `Bearer ${token}` } }).catch(() => null);
        if (!response?.ok && refresh) {
          const renewed = await fetch(`${apiBaseUrl}/api/v1/auth/refresh`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ refresh_token: refresh }),
          }).catch(() => null);
          if (renewed?.ok) {
            const session = (await renewed.json()) as { access_token: string; refresh_token: string };
            token = session.access_token;
            await SecureStore.setItemAsync('access_token', session.access_token);
            await SecureStore.setItemAsync('refresh_token', session.refresh_token);
          } else token = null;
        } else if (!response?.ok) token = null;
      }
      setAccessToken(token ?? '');
      setAuthLoaded(true);
    })();
  }, []);

  // 安卓返回键 / 侧边滑动：先逐层返回，再回到首页，只有首页才允许退出应用。
  useEffect(() => {
    const subscription = BackHandler.addEventListener('hardwareBackPress', () => {
      if (routes.length > 0) {
        setRoutes((prev) => prev.slice(0, -1));
        return true;
      }
      if (tab !== 'home') {
        setTab('home');
        return true;
      }
      return false;
    });
    return () => subscription.remove();
  }, [routes.length, tab]);

  const handleLogin = useCallback(async (token: string, refresh: string) => {
    queryClient.clear();
    await SecureStore.setItemAsync('access_token', token);
    await SecureStore.setItemAsync('refresh_token', refresh);
    setAccessToken(token);
    setTab('home');
    setRoutes([]);
  }, []);

  const handleLogout = useCallback(async () => {
    const refresh = await SecureStore.getItemAsync('refresh_token');
    if (refresh)
      await fetch(`${apiBaseUrl}/api/v1/auth/logout`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refresh }),
      }).catch(() => null);
    await SecureStore.deleteItemAsync('access_token');
    await SecureStore.deleteItemAsync('refresh_token');
    queryClient.clear();
    setAccessToken('');
    setRoutes([]);
    setTab('home');
  }, []);

  const push = (route: Route) => setRoutes((prev) => [...prev, route]);
  const pop = () => setRoutes((prev) => prev.slice(0, -1));

  if (!authLoaded)
    return (
      <SafeAreaView style={styles.safe}>
        <ActivityIndicator color={colors.green} style={{ marginTop: 40 }} />
      </SafeAreaView>
    );
  if (!accessToken) return <AuthScreen onLogin={handleLogin} />;

  const top = routes[routes.length - 1];
  let body: React.ReactNode;
  if (top?.kind === 'course') {
    body = <CourseDetail course={top.course} token={accessToken} onBack={pop} />;
  } else if (top?.kind === 'post') {
    body = (
      <PostDetailScreen
        token={accessToken}
        postId={top.postId}
        course={top.course}
        seed={top.seed}
        onBack={pop}
        onOpenCourse={(course) => push({ kind: 'course', course })}
      />
    );
  } else if (top?.kind === 'admin') {
    body = <AdminScreen token={accessToken} role="admin" onBack={pop} />;
  } else if (top?.kind === 'edit') {
    body = (
      <ProfileEditScreen
        token={accessToken}
        edit={top.edit}
        onBack={pop}
        onSaved={async () => {
          pop();
          await queryClient.invalidateQueries({ queryKey: ['my-reviews'] });
          await queryClient.invalidateQueries({ queryKey: ['my-posts'] });
        }}
      />
    );
  } else if (top?.kind === 'list') {
    body = <MyListScreen token={accessToken} kind={top.list} onBack={pop} onEdit={(edit) => push({ kind: 'edit', edit })} />;
  } else if (top?.kind === 'compose') {
    body = <ComposePostScreen token={accessToken} onClose={pop} onPublished={pop} />;
  } else if (tab === 'courses') {
    body = <CourseCatalog token={accessToken} onOpenCourse={(course) => push({ kind: 'course', course })} />;
  } else if (tab === 'share') {
    body = (
      <ShareFeedScreen
        token={accessToken}
        onOpenPost={(post) => push({ kind: 'post', postId: post.id, course: post.course, seed: post })}
        onCompose={() => push({ kind: 'compose' })}
      />
    );
  } else if (tab === 'notifications') {
    body = <NotificationsScreen token={accessToken} onBack={() => setTab('home')} />;
  } else if (tab === 'me') {
    body = (
      <MeScreen
        token={accessToken}
        onLogout={() => void handleLogout()}
        onOpenList={(list) => push({ kind: 'list', list })}
        onOpenAdmin={() => push({ kind: 'admin' })}
      />
    );
  } else {
    body = (
      <HomeScreen
        token={accessToken}
        onOpenCourse={(course) => push({ kind: 'course', course })}
        onOpenPost={(post) =>
          push({
            kind: 'post',
            postId: post.id,
            course: post.course,
            seed: {
              title: post.title,
              body: post.body,
              attendance_year: post.attendance_year,
              attendance_year_label: post.attendance_year_label,
              display_identity: post.display_identity,
              display_name: post.display_name,
              created_at: post.created_at,
              like_count: post.like_count,
              comment_count: post.comment_count,
            },
          })
        }
      />
    );
  }

  return (
    <SafeAreaView style={styles.safe}>
      <StatusBar style="dark" />
      <View style={styles.screen}>
        {routes.length === 0 && (
          <View style={styles.topbar}>
            <View style={styles.brand}>
              <View style={styles.mark}>
                <Text style={styles.markText}>C</Text>
              </View>
              <View>
                <Text style={styles.brandName}>CourseRep</Text>
                <Text style={styles.brandCaption}>CITYU · 课程经验社区</Text>
              </View>
            </View>
            <TouchableOpacity accessibilityLabel="通知" style={styles.bell} onPress={() => setTab('notifications')}>
              <Text style={styles.bellText}>♧</Text>
            </TouchableOpacity>
          </View>
        )}
        <View style={styles.body}>{body}</View>
        <View style={styles.bottomNav}>
          {tabs.map((item) => {
            const active = routes.length === 0 && tab === item.key;
            return (
              <TouchableOpacity
                key={item.key}
                style={styles.navItem}
                onPress={() => {
                  setRoutes([]);
                  setTab(item.key);
                }}
                accessibilityRole="button"
                accessibilityState={{ selected: active }}
              >
                <Text style={[styles.navIcon, active && styles.navSelected]}>{item.icon}</Text>
                <Text style={[styles.navLabel, active && styles.navSelected]}>{item.label}</Text>
              </TouchableOpacity>
            );
          })}
        </View>
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: colors.paper, paddingTop: NativeStatusBar.currentHeight ?? 0 },
  screen: { flex: 1, backgroundColor: colors.paper },
  body: { flex: 1 },
  topbar: {
    height: 58,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 20,
  },
  brand: { flexDirection: 'row', gap: 10, alignItems: 'center' },
  mark: { width: 36, height: 36, borderRadius: 13, backgroundColor: colors.green, alignItems: 'center', justifyContent: 'center' },
  markText: { color: colors.lime, fontSize: 20, fontWeight: '800' },
  brandName: { color: colors.ink, fontSize: 16, fontWeight: '800', letterSpacing: -0.4 },
  brandCaption: { color: '#7a827d', fontSize: 9, letterSpacing: 0.8, marginTop: 2 },
  bell: {
    width: 36,
    height: 36,
    borderRadius: 13,
    backgroundColor: '#fff',
    alignItems: 'center',
    justifyContent: 'center',
    borderWidth: 1,
    borderColor: colors.line,
  },
  bellText: { color: colors.green, fontSize: 18 },
  authContent: { flexGrow: 1, justifyContent: 'center', padding: 28, backgroundColor: colors.paper },
  authTitle: { color: colors.ink, fontSize: 26, fontWeight: '800', marginTop: 22 },
  authIntro: { color: '#78817a', fontSize: 13, marginTop: 8, marginBottom: 22 },
  authInput: {
    height: 50,
    backgroundColor: '#fff',
    borderWidth: 1,
    borderColor: colors.line,
    borderRadius: 14,
    paddingHorizontal: 14,
    color: colors.ink,
    marginBottom: 11,
  },
  authMessage: { color: '#8b513d', fontSize: 12, lineHeight: 18, marginVertical: 8 },
  authButton: { height: 50, backgroundColor: colors.green, borderRadius: 14, alignItems: 'center', justifyContent: 'center', marginTop: 8 },
  authButtonText: { color: '#fff', fontSize: 14, fontWeight: '700' },
  authSwitch: { color: colors.green, textAlign: 'center', fontSize: 13 },
  placeholder: { flexGrow: 1, justifyContent: 'center', padding: 28, gap: 10 },
  placeholderTitle: { color: colors.ink, fontSize: 18, fontWeight: '800' },
  placeholderBody: { color: colors.muted, fontSize: 12, lineHeight: 19 },
  primary: { alignSelf: 'flex-start', backgroundColor: colors.green, borderRadius: 12, paddingHorizontal: 18, paddingVertical: 11, marginTop: 6 },
  primaryText: { color: '#fff', fontSize: 12, fontWeight: '800' },
  bottomNav: {
    minHeight: 64,
    backgroundColor: '#fffffff2',
    borderTopWidth: 1,
    borderTopColor: colors.line,
    flexDirection: 'row',
    justifyContent: 'space-around',
    alignItems: 'center',
    paddingBottom: 4,
  },
  navItem: { minWidth: 50, alignItems: 'center', gap: 3 },
  navIcon: { color: '#939b94', fontSize: 17, height: 21 },
  navLabel: { color: '#939b94', fontSize: 9 },
  navSelected: { color: colors.green, fontWeight: '800' },
});
