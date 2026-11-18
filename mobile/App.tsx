import { StatusBar } from 'expo-status-bar';
import { useCallback, useEffect, useRef, useState } from 'react';
import * as SecureStore from 'expo-secure-store';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { CourseDetail } from './CourseDetail';
import { confirmDiscard, type LeaveState } from './src/ui/unsavedChanges';
import type { CourseDetailCourse } from './src/api/types';
import { AdminScreen, NotificationsScreen, ProfileEditScreen } from './UserScreens';
import { apiBaseUrl, apiFetch, fetchWithTimeout, initApiBase, isTimeoutError } from './api';
import type {
  ForgotPasswordRequest,
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  ResetPasswordRequest,
  VerifyEmailRequest,
} from './src/api/types';
import { CourseCatalog } from './src/screens/CourseCatalog';
import { ComposePostScreen } from './src/screens/ComposePostScreen';
import { HomeScreen } from './src/screens/HomeScreen';
import { MeScreen } from './src/screens/MeScreen';
import { MyListScreen, type EditTarget, type MyListKind } from './src/screens/MyListScreen';
import { PostDetailScreen, type PostCoursePreview } from './src/screens/PostDetailScreen';
import { ShareFeedScreen } from './src/screens/ShareFeedScreen';
import { ServerField } from './src/ui/ServerField';
import { colors } from './src/ui/theme';
import {
  ActivityIndicator,
  BackHandler,
  Keyboard,
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
  | { kind: 'course'; courseId: number }
  | { kind: 'post'; postId: number; course?: PostCoursePreview }
  | { kind: 'list'; list: MyListKind; focusId?: number }
  | { kind: 'admin' }
  | { kind: 'compose'; course?: CourseDetailCourse }
  | { kind: 'edit'; edit: EditTarget };

const tabs: { key: Tab; icon: string; label: string }[] = [
  { key: 'home', icon: '⌂', label: '首页' },
  { key: 'courses', icon: '▤', label: '课程' },
  { key: 'share', icon: '＋', label: '分享' },
  { key: 'notifications', icon: '♧', label: '通知' },
  { key: 'me', icon: '◉', label: '我的' },
];

const queryClient = new QueryClient({ defaultOptions: { queries: { staleTime: 15_000, retry: 1 } } });

const authCopy = {
  login: { title: '登录 CourseRep', intro: '使用学校邮箱登录或注册', submit: '登录' },
  register: { title: '创建校内账号', intro: '使用学校邮箱登录或注册', submit: '发送验证码' },
  verify: { title: '验证学生邮箱', intro: '验证码有效期为 10 分钟，也可检查垃圾邮件文件夹。', submit: '验证邮箱' },
  'forgot-password': { title: '找回密码', intro: '输入注册时使用的学生邮箱。', submit: '发送验证码' },
  'reset-password': { title: '设置新密码', intro: '输入邮件里的 6 位验证码，验证码有效期为 10 分钟。', submit: '重置密码' },
};

function AuthScreen({ onLogin }: { onLogin: (access: string, refresh: string) => Promise<void> }) {
  const [mode, setMode] = useState<keyof typeof authCopy>('login');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [nickname, setNickname] = useState('');
  const [code, setCode] = useState('');
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState('');
  const [resendAt, setResendAt] = useState(0);
  const [now, setNow] = useState(Date.now);
  const waitingForCode = mode === 'verify' || mode === 'reset-password';
  const resendIn = Math.max(0, Math.ceil((resendAt - now) / 1000));
  useEffect(() => {
    if (!waitingForCode || resendIn === 0) return;
    const timer = setTimeout(() => setNow(Date.now()), 1000);
    return () => clearTimeout(timer);
  }, [waitingForCode, resendIn, now]);

  const submit = async (resend = false) => {
    if (busy || (resend && resendIn > 0)) return;
    if (mode === 'reset-password' && !resend && password !== confirmPassword) {
      setMessage('两次输入的新密码不一致');
      return;
    }
    setBusy(true);
    setMessage('');
    try {
      const path = resend ? (mode === 'verify' ? 'register' : 'forgot-password') : mode;
      const bodies: {
        login: LoginRequest;
        register: RegisterRequest;
        verify: VerifyEmailRequest;
        'forgot-password': ForgotPasswordRequest;
        'reset-password': ResetPasswordRequest;
      } = {
        login: { email, password },
        register: { email, password, nickname },
        verify: { email, code },
        'forgot-password': { email },
        'reset-password': { email, code, password },
      };
      const response = await fetchWithTimeout(
        `${apiBaseUrl}/api/v1/auth/${path}`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(bodies[path]),
        },
        15000,
      );
      const payload = (await response.json()) as LoginResponse & { error?: string; message?: string };
      if (!response.ok) {
        const retryAfter = Number(response.headers.get('Retry-After'));
        if (resend && response.status === 429) {
          const current = Date.now();
          setNow(current);
          setResendAt(current + (retryAfter > 0 ? retryAfter : 60) * 1000);
        }
        throw new Error(payload.error ?? '请求失败，请稍后再试');
      }
      if (path === 'login') {
        if (!payload.access_token || !payload.refresh_token) throw new Error('登录服务返回异常，请稍后重试');
        await onLogin(payload.access_token, payload.refresh_token);
      } else if (path === 'register' || path === 'forgot-password') {
        const current = Date.now();
        setNow(current);
        setResendAt(current + 60_000);
        setCode('');
        setMode(path === 'register' ? 'verify' : 'reset-password');
        setMessage(payload.message ?? '验证码已发送');
      } else {
        if (path === 'reset-password') {
          setPassword('');
          setConfirmPassword('');
          setCode('');
        }
        setMode('login');
        setMessage(path === 'verify' ? '邮箱验证完成，请登录' : '密码已重置，请使用新密码登录');
      }
    } catch (error) {
      setMessage(isTimeoutError(error) ? '服务暂时未响应，请检查网络后重试' : error instanceof TypeError ? '无法连接服务，请检查网络后重试' : error instanceof Error ? error.message : '暂时无法连接服务');
    } finally {
      setBusy(false);
    }
  };
  const { title, intro, submit: submitLabel } = authCopy[mode];
  return (
    <SafeAreaView style={styles.safe}>
      <ScrollView contentContainerStyle={styles.authContent} keyboardShouldPersistTaps="handled">
        <View style={styles.mark}>
          <Text style={styles.markText}>C</Text>
        </View>
        <Text style={styles.authTitle}>{title}</Text>
        <Text style={styles.authIntro}>{intro}</Text>
        <TextInput
          autoCapitalize="none"
          keyboardType="email-address"
          value={email}
          onChangeText={setEmail}
          editable={!busy && !waitingForCode}
          placeholder="学生邮箱"
          style={styles.authInput}
        />
        {waitingForCode && (
          <TextInput value={code} onChangeText={setCode} editable={!busy} keyboardType="number-pad" autoComplete="one-time-code" maxLength={6} placeholder="邮箱验证码" style={styles.authInput} />
        )}
        {(mode === 'login' || mode === 'register' || mode === 'reset-password') && (
          <TextInput
            value={password}
            onChangeText={setPassword}
            editable={!busy}
            secureTextEntry
            autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
            placeholder={mode === 'reset-password' ? '新密码（至少 10 个字节）' : '密码（至少 10 位）'}
            style={styles.authInput}
          />
        )}
        {mode === 'reset-password' && (
          <TextInput value={confirmPassword} onChangeText={setConfirmPassword} editable={!busy} secureTextEntry autoComplete="new-password" placeholder="确认新密码" style={styles.authInput} />
        )}
        {mode === 'register' && <TextInput value={nickname} onChangeText={setNickname} editable={!busy} placeholder="昵称" style={styles.authInput} />}
        {!!message && <Text style={styles.authMessage}>{message}</Text>}
        <TouchableOpacity disabled={busy} onPress={() => void submit()} style={styles.authButton}>
          <Text style={styles.authButtonText}>
            {busy ? '请稍候…' : submitLabel}
          </Text>
        </TouchableOpacity>
        <TouchableOpacity
          disabled={busy}
          onPress={() => {
            if (mode === 'reset-password') {
              setPassword('');
              setConfirmPassword('');
              setCode('');
            }
            setMessage('');
            setMode(mode === 'login' ? 'register' : 'login');
          }}
          style={{ padding: 16 }}
        >
          <Text style={styles.authSwitch}>{mode === 'login' ? '还没有账号？注册' : '返回登录'}</Text>
        </TouchableOpacity>
        {mode === 'login' && (
          <TouchableOpacity
            accessibilityRole="button"
            disabled={busy}
            onPress={() => {
              setPassword('');
              setConfirmPassword('');
              setMessage('');
              setMode('forgot-password');
            }}
            style={{ padding: 14 }}
          >
            <Text style={styles.authSwitch}>忘记密码？</Text>
          </TouchableOpacity>
        )}
        {waitingForCode && (
          <>
            <TouchableOpacity disabled={busy || resendIn > 0} onPress={() => void submit(true)} style={{ padding: 14 }}>
              <Text style={styles.authSwitch}>{resendIn ? `${resendIn} 秒后可重发验证码` : '重新发送验证码'}</Text>
            </TouchableOpacity>
            <TouchableOpacity disabled={busy} onPress={() => { setMessage(''); setMode(mode === 'verify' ? 'register' : 'forgot-password'); }} style={{ padding: 14 }}>
              <Text style={styles.authSwitch}>{mode === 'verify' ? '修改注册信息' : '修改邮箱'}</Text>
            </TouchableOpacity>
          </>
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
  const [authState, setAuthState] = useState<'loading' | 'ready' | 'unavailable'>('loading');
  const [accessToken, setAccessToken] = useState('');
  const [tab, setTab] = useState<Tab>('home');
  const [visited, setVisited] = useState<Tab[]>(['home']);
  const [routes, setRoutes] = useState<Route[]>([]);
  const dirty = useRef<LeaveState>(false);
  const setDirty = useCallback((value: LeaveState) => { dirty.current = value; }, []);

  const restoreSession = useCallback(async () => {
    setAuthState('loading');
    try {
      await initApiBase();
      const token = await SecureStore.getItemAsync('access_token');
      const refresh = await SecureStore.getItemAsync('refresh_token');
      if (token) {
        const response = await fetchWithTimeout(`${apiBaseUrl}/api/v1/auth/me`, {
          headers: { Authorization: `Bearer ${token}` },
        });
        if (response.ok) {
          setAccessToken(token);
          setAuthState('ready');
          return;
        }
        if (response.status !== 401 && response.status !== 403) throw new Error('Session check failed');
      }
      if (refresh) {
        const response = await fetchWithTimeout(`${apiBaseUrl}/api/v1/auth/refresh`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: refresh }),
        });
        if (response.ok) {
          const session = (await response.json()) as LoginResponse;
          await SecureStore.setItemAsync('access_token', session.access_token);
          await SecureStore.setItemAsync('refresh_token', session.refresh_token);
          setAccessToken(session.access_token);
        } else if (response.status === 401 || response.status === 403) {
          await SecureStore.deleteItemAsync('access_token');
          await SecureStore.deleteItemAsync('refresh_token');
        } else {
          throw new Error('Session refresh failed');
        }
      } else if (token) {
        await SecureStore.deleteItemAsync('access_token');
      }
      setAuthState('ready');
    } catch {
      setAuthState('unavailable');
    }
  }, []);

  useEffect(() => { void restoreSession(); }, [restoreSession]);

  const activateTab = useCallback((next: Tab) => {
    setVisited((previous) => previous.includes(next) ? previous : [...previous, next]);
    setTab(next);
  }, []);

  const pop = useCallback(() => {
    Keyboard.dismiss();
    setRoutes((previous) => previous.slice(0, -1));
  }, []);
  const back = useCallback(() => confirmDiscard(dirty.current, pop), [pop]);

  useEffect(() => {
    const subscription = BackHandler.addEventListener('hardwareBackPress', () => {
      if (routes.length > 0) {
        back();
        return true;
      }
      if (tab !== 'home') {
        activateTab('home');
        return true;
      }
      return false;
    });
    return () => subscription.remove();
  }, [routes.length, tab, back, activateTab]);

  const handleLogin = useCallback(async (token: string, refresh: string) => {
    queryClient.clear();
    await SecureStore.setItemAsync('access_token', token);
    await SecureStore.setItemAsync('refresh_token', refresh);
    setAccessToken(token);
    setTab('home');
    setVisited(['home']);
    setRoutes([]);
  }, []);

  const handleLogout = useCallback(async () => {
    const refresh = await SecureStore.getItemAsync('refresh_token');
    if (refresh)
      await fetchWithTimeout(
        `${apiBaseUrl}/api/v1/auth/logout`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: refresh }),
        },
        5000,
      ).catch(() => null);
    await SecureStore.deleteItemAsync('access_token');
    await SecureStore.deleteItemAsync('refresh_token');
    queryClient.clear();
    setAccessToken('');
    setRoutes([]);
    setTab('home');
    setVisited(['home']);
  }, []);

  const push = (route: Route) => {
    Keyboard.dismiss();
    setRoutes((previous) => [...previous, route]);
  };
  const openCourse = (courseId: number) => push({ kind: 'course', courseId });

  if (authState === 'loading')
    return (
      <SafeAreaView style={styles.safe}>
        <ActivityIndicator color={colors.green} style={{ marginTop: 40 }} />
      </SafeAreaView>
    );
  if (authState === 'unavailable') {
    return (
      <SafeAreaView style={styles.safe}>
        <View style={styles.placeholder}>
          <Text style={styles.placeholderTitle}>暂时无法恢复登录</Text>
          <Text style={styles.placeholderBody}>请检查网络连接后重试。</Text>
          <TouchableOpacity style={styles.primary} onPress={() => void restoreSession()}>
            <Text style={styles.primaryText}>重试</Text>
          </TouchableOpacity>
          <ServerField />
        </View>
      </SafeAreaView>
    );
  }
  if (!accessToken) return <AuthScreen onLogin={handleLogin} />;

  const chooseTab = (next: Tab) => {
    if (next === tab && routes.length === 0) return;
    confirmDiscard(dirty.current, () => {
      Keyboard.dismiss();
      setRoutes([]);
      activateTab(next);
    });
  };

  const openPost = (postId: number, course?: PostCoursePreview) => push({ kind: 'post', postId, course });

  const renderRoute = (route: Route) => {
    switch (route.kind) {
      case 'course':
        return <CourseDetail courseId={route.courseId} token={accessToken} onBack={back}
          onCompose={(course) => push({ kind: 'compose', course })} onDirtyChange={setDirty} />;
      case 'post':
        return <PostDetailScreen token={accessToken} postId={route.postId} course={route.course}
          onBack={back} onOpenCourse={openCourse} />;
      case 'admin':
        return <AdminScreen token={accessToken} role="admin" onBack={back} />;
      case 'edit':
        return <ProfileEditScreen token={accessToken} edit={route.edit} onBack={back} onDirtyChange={setDirty}
          onSaved={async () => {
            pop();
            await Promise.all([
              queryClient.invalidateQueries({ queryKey: ['my-reviews'] }),
              queryClient.invalidateQueries({ queryKey: ['my-posts'] }),
              queryClient.invalidateQueries({ queryKey: ['posts'] }),
              queryClient.invalidateQueries({ queryKey: ['reviews'] }),
              queryClient.invalidateQueries({ queryKey: ['course'] }),
              queryClient.invalidateQueries({ queryKey: ['catalog'] }),
              queryClient.invalidateQueries({ queryKey: ['post-detail'] }),
              queryClient.invalidateQueries({ queryKey: ['feed'] }),
              queryClient.invalidateQueries({ queryKey: ['home'] }),
            ]);
          }} />;
      case 'list':
        return <MyListScreen token={accessToken} kind={route.list} focusId={route.focusId} onBack={back}
          onEdit={(edit) => push({ kind: 'edit', edit })}
          onOpenBookmark={(item) => openPost(item.id, item)} />;
      case 'compose':
        return <ComposePostScreen token={accessToken} initialCourse={route.course} onClose={back}
          onPublished={pop} onDirtyChange={setDirty} />;
    }
  };

  const renderTab = (current: Tab) => {
    switch (current) {
      case 'courses':
        return <CourseCatalog token={accessToken} onOpenCourse={(course) => openCourse(course.id)} />;
      case 'share':
        return <ShareFeedScreen token={accessToken} onOpenPost={(post) => openPost(post.id, post.course)}
          onCompose={() => push({ kind: 'compose' })} />;
      case 'notifications':
        return <NotificationsScreen token={accessToken} active={tab === current && routes.length === 0} onBack={() => activateTab('home')} onOpenPost={openPost}
          onOpenOwnContent={(list, focusId) => push({ kind: 'list', list, focusId })} />;
      case 'me':
        return <MeScreen token={accessToken} onLogout={() => void handleLogout()}
          onOpenList={(list) => push({ kind: 'list', list })}
          onOpenAdmin={() => push({ kind: 'admin' })} />;
      case 'home':
        return <HomeScreen token={accessToken} onOpenCourse={(course) => openCourse(course.id)}
          onOpenPost={(post) => openPost(post.id, post.course)} />;
    }
  };

  return (
    <SafeAreaView style={styles.safe}>
      <StatusBar style="dark" />
      <View style={styles.screen}>
        <View style={styles.body}>
          <ScreenLayer visible={routes.length === 0}>
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
              <TouchableOpacity accessibilityLabel="通知" style={styles.bell} onPress={() => chooseTab('notifications')}>
                <Text style={styles.bellText}>♧</Text>
              </TouchableOpacity>
            </View>
            <View style={styles.body}>
              {visited.map((current) => (
                <ScreenLayer key={current} visible={tab === current}>
                  {renderTab(current)}
                </ScreenLayer>
              ))}
            </View>
          </ScreenLayer>
          {routes.map((route, index) => (
            <ScreenLayer key={`${index}-${route.kind}`} visible={index === routes.length - 1}>
              {renderRoute(route)}
            </ScreenLayer>
          ))}
        </View>
        <View style={styles.bottomNav}>
          {tabs.map((item) => {
            const active = routes.length === 0 && tab === item.key;
            return (
              <TouchableOpacity
                key={item.key}
                style={styles.navItem}
                onPress={() => chooseTab(item.key)}
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

function ScreenLayer({ visible, children }: { visible: boolean; children: React.ReactNode }) {
  return (
    <View style={[StyleSheet.absoluteFill, !visible && { opacity: 0 }]}
      pointerEvents={visible ? 'auto' : 'none'}
      accessibilityElementsHidden={!visible}
      importantForAccessibility={visible ? 'auto' : 'no-hide-descendants'}>
      {children}
    </View>
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
