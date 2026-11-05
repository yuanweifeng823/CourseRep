import { StatusBar } from 'expo-status-bar';
import { useCallback, useEffect, useState } from 'react';
import * as SecureStore from 'expo-secure-store';
import { QueryClient, QueryClientProvider, useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { CourseDetail } from './CourseDetail';
import { NotificationsScreen, ProfileScreen } from './UserScreens';
import { apiFetch } from './api';
import type { Course, CourseFacets, CourseListResponse, LoginRequest, LoginResponse, RegisterRequest } from './src/api/types';
import {
  ActivityIndicator,
  Platform,
  RefreshControl,
  SafeAreaView,
  ScrollView,
  StatusBar as NativeStatusBar,
  StyleSheet,
  Text,
  TextInput,
  TouchableOpacity,
  View,
} from 'react-native';

const tabs = [
  { icon: '⌂', label: '发现' },
  { icon: '▤', label: '课程' },
  { icon: '＋', label: '分享' },
  { icon: '♧', label: '通知' },
  { icon: '◉', label: '我的' },
];

const apiBaseUrl = (process.env.EXPO_PUBLIC_API_URL ||
  (Platform.OS === 'android' ? 'http://10.0.2.2:8080' : 'http://localhost:8080')).replace(/\/$/, '');
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
    setBusy(true); setMessage('');
    try {
      const path = mode === 'verify' ? 'verify' : mode;
      const body: LoginRequest | RegisterRequest | { email: string; code: string } = mode === 'login' ? { email, password } : mode === 'register' ? { email, password, nickname } : { email, code };
      const response = await fetch(`${apiBaseUrl}/api/v1/auth/${path}`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
      const payload = await response.json() as LoginResponse & { error?: string; message?: string };
      if (!response.ok) throw new Error(payload.error ?? '请求失败，请稍后再试');
      if (mode === 'login' && payload.access_token && payload.refresh_token) await onLogin(payload.access_token, payload.refresh_token);
      else if (mode === 'register') { setMode('verify'); setMessage(payload.message ?? '验证码已发送'); }
      else { setMode('login'); setMessage('邮箱验证完成，请登录'); }
    } catch (error) { setMessage(error instanceof Error ? error.message : '暂时无法连接服务'); }
    finally { setBusy(false); }
  };
  const title = mode === 'login' ? '登录 CourseRep' : mode === 'register' ? '创建校内账号' : '验证学生邮箱';
  return <SafeAreaView style={styles.safe}><ScrollView contentContainerStyle={styles.authContent} keyboardShouldPersistTaps="handled">
    <View style={styles.mark}><Text style={styles.markText}>C</Text></View><Text style={styles.authTitle}>{title}</Text>
    <Text style={styles.authIntro}>使用学校允许的学生邮箱域名注册</Text>
    <TextInput autoCapitalize="none" keyboardType="email-address" value={email} onChangeText={setEmail} placeholder="学生邮箱" style={styles.authInput} />
    {mode !== 'verify' && <TextInput value={password} onChangeText={setPassword} secureTextEntry placeholder="密码（至少 10 位）" style={styles.authInput} />}
    {mode === 'register' && <TextInput value={nickname} onChangeText={setNickname} placeholder="昵称" style={styles.authInput} />}
    {mode === 'verify' && <TextInput value={code} onChangeText={setCode} keyboardType="number-pad" maxLength={6} placeholder="邮箱验证码" style={styles.authInput} />}
    {!!message && <Text style={styles.authMessage}>{message}</Text>}
    <TouchableOpacity disabled={busy} onPress={() => void submit()} style={styles.authButton}><Text style={styles.authButtonText}>{busy ? '请稍候…' : mode === 'login' ? '登录' : mode === 'register' ? '发送验证码' : '验证邮箱'}</Text></TouchableOpacity>
    <TouchableOpacity onPress={() => { setMessage(''); setMode(mode === 'login' ? 'register' : 'login'); }} style={{ padding: 16 }}><Text style={styles.authSwitch}>{mode === 'login' ? '还没有账号？注册' : '返回登录'}</Text></TouchableOpacity>
    {mode === 'verify' && <TouchableOpacity onPress={() => setMode('register')}><Text style={styles.authSwitch}>修改注册信息</Text></TouchableOpacity>}
  </ScrollView></SafeAreaView>;
}

export default function App() {
  return <QueryClientProvider client={queryClient}><AppShell /></QueryClientProvider>;
}

function AppShell() {
  const [authLoaded, setAuthLoaded] = useState(false);
  const [accessToken, setAccessToken] = useState('');
  const [activeTab, setActiveTab] = useState('发现');
  const [selectedCourse, setSelectedCourse] = useState<Course | null>(null);
  const [query, setQuery] = useState('');
  const [department,setDepartment]=useState('');const [semester,setSemester]=useState('');const [sort,setSort]=useState<'course'|'rating'|'reviews'|'posts'>('course');const [debouncedQuery,setDebouncedQuery]=useState('');
  useEffect(() => {
    void (async () => {
      let token = await SecureStore.getItemAsync('access_token');
      const refresh = await SecureStore.getItemAsync('refresh_token');
      if (token) {
        const response = await fetch(`${apiBaseUrl}/api/v1/auth/me`, { headers: { Authorization: `Bearer ${token}` } }).catch(() => null);
        if (!response?.ok && refresh) {
          const renewed = await fetch(`${apiBaseUrl}/api/v1/auth/refresh`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ refresh_token: refresh }) }).catch(() => null);
          if (renewed?.ok) {
            const session = await renewed.json() as { access_token: string; refresh_token: string };
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

  useEffect(()=>{const timer=setTimeout(()=>setDebouncedQuery(query.trim()),300);return()=>clearTimeout(timer)},[query]);
  const courseSearch=useInfiniteQuery({queryKey:['courses',!!accessToken,debouncedQuery,department,semester,sort],enabled:!!accessToken,initialPageParam:1,queryFn:async({pageParam})=>{const params=new URLSearchParams({page:String(pageParam),page_size:'100',q:debouncedQuery,sort});if(department)params.set('department',department);if(semester)params.set('semester',semester);const response=await apiFetch(`/api/v1/courses?${params.toString()}`,accessToken,{headers:{Accept:'application/json'}});if(!response.ok)throw new Error('暂时无法连接课程服务');const payload=await response.json() as CourseListResponse;if(!payload||!Array.isArray(payload.items))throw new Error('课程接口响应格式不正确');return payload;},getNextPageParam:(lastPage)=>{const count=lastPage.items?.length??0;return count>0&&count===(lastPage.page_size??100)?(lastPage.page??1)+1:undefined}});
  const facetQuery=useQuery({queryKey:['course-facets',!!accessToken],enabled:!!accessToken,queryFn:async()=>{const r=await apiFetch('/api/v1/course-facets',accessToken);if(!r.ok)throw new Error('读取课程筛选项失败');return await r.json() as CourseFacets}});
  const courses=(courseSearch.data?.pages.flatMap(page=>page.items??[])??[]).filter((item):item is NonNullable<typeof item>=>item!==null);const loading=courseSearch.isLoading||courseSearch.isFetching;const apiError=courseSearch.error?`暂时无法连接课程服务（${apiBaseUrl}）。请确认后端正在运行，并检查 API 地址。`:'';const facets={departments:facetQuery.data?.departments??[],semesters:facetQuery.data?.semesters??[]};const loadCourses=()=>void courseSearch.refetch();

  const handleLogin = useCallback(async (token: string, refresh: string) => {
    queryClient.clear();
    await SecureStore.setItemAsync('access_token', token);
    await SecureStore.setItemAsync('refresh_token', refresh);
    setAccessToken(token);
  }, []);
  const handleLogout = useCallback(async () => {
    const refresh = await SecureStore.getItemAsync('refresh_token');
    if (refresh) await fetch(`${apiBaseUrl}/api/v1/auth/logout`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ refresh_token: refresh }) }).catch(() => null);
    await SecureStore.deleteItemAsync('access_token');
    await SecureStore.deleteItemAsync('refresh_token');
    queryClient.clear();
    setAccessToken('');
  }, []);

  if (!authLoaded) return <SafeAreaView style={styles.safe}><ActivityIndicator color={green} /></SafeAreaView>;
  if (!accessToken) return <AuthScreen onLogin={handleLogin} />;
  if (selectedCourse) return <CourseDetail course={selectedCourse} token={accessToken} onBack={() => setSelectedCourse(null)} />;
  if (activeTab === '通知') return <NotificationsScreen token={accessToken} onBack={() => setActiveTab('发现')} />;
  if (activeTab === '我的') return <ProfileScreen token={accessToken} onBack={() => setActiveTab('发现')} onLogout={() => void handleLogout()} />;
  if (activeTab === '分享') return <SafeAreaView style={styles.safe}><View style={styles.authContent}><Text style={styles.sectionTitle}>先选择课程</Text><Text style={styles.emptyBody}>经验帖必须关联课程。打开课程详情后，可以发布修课经验。</Text><TouchableOpacity style={styles.retry} onPress={() => setActiveTab('课程')}><Text style={styles.retryText}>浏览课程</Text></TouchableOpacity></View></SafeAreaView>;

  const filteredTitle = query.trim();

  return (
    <SafeAreaView style={styles.safe}>
      <StatusBar style="dark" />
      <View style={styles.screen}>
        <ScrollView
          contentContainerStyle={styles.content}
          keyboardShouldPersistTaps="handled"
          refreshControl={<RefreshControl refreshing={loading} onRefresh={loadCourses} tintColor={green} />}
        >
          <View style={styles.topbar}>
            <View style={styles.brand}>
              <View style={styles.mark}><Text style={styles.markText}>C</Text></View>
              <View>
                <Text style={styles.brandName}>CourseRep</Text>
                <Text style={styles.brandCaption}>CITYU · 课程经验社区</Text>
              </View>
            </View>
            <TouchableOpacity accessibilityLabel="通知" style={styles.bell} onPress={() => setActiveTab('通知')}>
              <Text style={styles.bellText}>♧</Text>
            </TouchableOpacity>
          </View>

          <Text style={styles.eyebrow}>COURSE DISCOVERY</Text>
          <Text style={styles.title}>找到适合你的{ '\n' }下一门课</Text>
          <Text style={styles.intro}>搜索课程，看看同学们真实的修读体验。</Text>

          <View style={styles.searchBox}>
            <Text style={styles.searchIcon}>⌕</Text>
            <TextInput
              accessibilityLabel="搜索课程代码、名称或教授"
              placeholder="课程代码、课程名称或教授"
              placeholderTextColor="#929a93"
              value={query}
              onChangeText={setQuery}
              style={styles.searchInput}
              returnKeyType="search"
            />
            {query.length > 0 && <TouchableOpacity onPress={() => setQuery('')}><Text style={styles.clear}>清除</Text></TouchableOpacity>}
          </View>

          <Text style={styles.filterHeading}>排序方式</Text><ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.filters}>{([{key:'course',label:'课程代码'},{key:'rating',label:'评分'},{key:'reviews',label:'评价数'},{key:'posts',label:'分享数'}] as const).map(x=><TouchableOpacity key={x.key} onPress={()=>setSort(x.key)} style={[styles.filter,sort===x.key&&styles.filterActive]}><Text style={[styles.filterText,sort===x.key&&styles.filterTextActive]}>{x.label}</Text></TouchableOpacity>)}</ScrollView>
          <Text style={styles.filterHeading}>所属专业</Text><ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.filters}><TouchableOpacity onPress={()=>setDepartment('')} style={[styles.filter,!department&&styles.filterActive]}><Text style={[styles.filterText,!department&&styles.filterTextActive]}>全部</Text></TouchableOpacity>{facets.departments.map(x=><TouchableOpacity key={x} onPress={()=>setDepartment(department===x?'':x)} style={[styles.filter,department===x&&styles.filterActive]}><Text style={[styles.filterText,department===x&&styles.filterTextActive]}>{x}</Text></TouchableOpacity>)}</ScrollView>
          <Text style={styles.filterHeading}>学期</Text><ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.filters}><TouchableOpacity onPress={()=>setSemester('')} style={[styles.filter,!semester&&styles.filterActive]}><Text style={[styles.filterText,!semester&&styles.filterTextActive]}>全部</Text></TouchableOpacity>{facets.semesters.map(x=><TouchableOpacity key={x} onPress={()=>setSemester(semester===x?'':x)} style={[styles.filter,semester===x&&styles.filterActive]}><Text style={[styles.filterText,semester===x&&styles.filterTextActive]}>{x}</Text></TouchableOpacity>)}</ScrollView>

          <View style={styles.sectionHeader}>
            <View>
              <Text style={styles.sectionTitle}>课程目录</Text>
              <Text style={styles.sectionSub}>从真实课程资料开始探索</Text>
            </View>
            <Text style={styles.count}>{courses.length} 门课程</Text>
          </View>
          <Text style={styles.sectionSub}>搜索范围：课程代码、名称、教授和专业</Text>

          {loading && courses.length === 0 && !apiError ? (
            <View style={styles.emptyCard}>
              <ActivityIndicator color={green} />
              <Text style={styles.emptyTitle}>正在读取课程目录</Text>
              <Text style={styles.emptyBody}>正在连接 CourseRep 服务…</Text>
            </View>
          ) : apiError ? (
            <View style={styles.emptyCard}>
              <View style={styles.emptyIcon}><Text style={styles.errorIcon}>!</Text></View>
              <Text style={styles.emptyTitle}>课程服务暂不可用</Text>
              <Text style={styles.emptyBody}>{apiError}</Text>
              <TouchableOpacity style={styles.retry} onPress={loadCourses}><Text style={styles.retryText}>重新连接</Text></TouchableOpacity>
            </View>
          ) : courses.length === 0 ? (
            <View style={styles.emptyCard}>
              <View style={styles.emptyIcon}><Text style={styles.emptyIconText}>⌕</Text></View>
              <Text style={styles.emptyTitle}>{filteredTitle ? '暂时没有匹配课程' : '课程目录正在准备中'}</Text>
              <Text style={styles.emptyBody}>
                {filteredTitle
                  ? `没有找到“${filteredTitle}”。课程资料导入后，你可以在这里搜索和筛选。`
                  : '课程资料经校对后会陆续加入。现在还没有课程数据，我们不会用示例课程冒充真实目录。'}
              </Text>
              <View style={styles.emptyDivider} />
              <Text style={styles.emptyFoot}>课程代码 · 教授 · 专业 · 学期</Text>
            </View>
          ) : (
            <View style={styles.courseList}>
              {courses.map((course) => (
                <TouchableOpacity key={course.id} style={styles.courseCard} onPress={() => setSelectedCourse(course)}>
                  <View style={styles.courseMark}><Text style={styles.courseMarkText}>{course.course_code.slice(0, 1)}</Text></View>
                  <View style={styles.courseInfo}>
                    <Text style={styles.courseCode}>{course.course_code} · {course.semester}</Text>
                    <Text style={styles.courseName}>{course.course_name}</Text>
                    <Text style={styles.courseMeta}>{course.professor || '教授待补充'} · {course.department} · {course.credits} 学分</Text><Text style={styles.courseMeta}>评分 {course.rating.toFixed(1)} · {course.review_count} 条评价 · {course.post_count} 篇分享</Text>
                  </View>
                </TouchableOpacity>
              ))}
            </View>
          )}
          {courses.length>0&&courseSearch.hasNextPage&&<TouchableOpacity disabled={courseSearch.isFetchingNextPage} style={styles.loadMore} onPress={()=>void courseSearch.fetchNextPage()}><Text style={styles.loadMoreText}>{courseSearch.isFetchingNextPage?'正在加载…':'加载更多课程'}</Text></TouchableOpacity>}

          <View style={styles.infoCard}>
            <View style={styles.infoDot}><Text style={styles.infoDotText}>✦</Text></View>
            <View style={styles.infoCopy}>
              <Text style={styles.infoTitle}>分享你的修课经验</Text>
              <Text style={styles.infoBody}>目录开放后，课程评价和经验帖会在对应课程下展示。</Text>
            </View>
            <Text style={styles.infoArrow}>›</Text>
          </View>
          <Text style={styles.footer}>为 CityU 同学整理真实、可信的课程信息</Text>
          <TouchableOpacity onPress={() => void handleLogout()} style={{ alignSelf: 'center', padding: 14 }}><Text style={{ color: green, fontSize: 12 }}>退出登录</Text></TouchableOpacity>
        </ScrollView>

        <View style={styles.bottomNav}>
          {tabs.map((tab) => (
            <TouchableOpacity key={tab.label} style={styles.navItem} onPress={() => setActiveTab(tab.label)}>
              <Text style={[styles.navIcon, activeTab === tab.label && styles.navSelected]}>{tab.icon}</Text>
              <Text style={[styles.navLabel, activeTab === tab.label && styles.navSelected]}>{tab.label}</Text>
            </TouchableOpacity>
          ))}
        </View>
      </View>
    </SafeAreaView>
  );
}

const green = '#315f4a';
const ink = '#202722';
const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: '#f5f6f2', paddingTop: NativeStatusBar.currentHeight ?? 0 },
  screen: { flex: 1, backgroundColor: '#f5f6f2' },
  content: { paddingHorizontal: 22, paddingBottom: 34 },
  authContent: { flexGrow: 1, justifyContent: 'center', padding: 28, backgroundColor: '#f5f6f2' },
  authTitle: { color: ink, fontSize: 26, fontWeight: '800', marginTop: 22 },
  authIntro: { color: '#78817a', fontSize: 13, marginTop: 8, marginBottom: 22 },
  authInput: { height: 50, backgroundColor: '#fff', borderWidth: 1, borderColor: '#e9ece6', borderRadius: 14, paddingHorizontal: 14, color: ink, marginBottom: 11 },
  authMessage: { color: '#8b513d', fontSize: 12, lineHeight: 18, marginVertical: 8 },
  authButton: { height: 50, backgroundColor: green, borderRadius: 14, alignItems: 'center', justifyContent: 'center', marginTop: 8 },
  authButtonText: { color: '#fff', fontSize: 14, fontWeight: '700' },
  authSwitch: { color: green, textAlign: 'center', fontSize: 13 },
  topbar: { height: 62, flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', marginBottom: 25 },
  brand: { flexDirection: 'row', gap: 10, alignItems: 'center' },
  mark: { width: 39, height: 39, borderRadius: 14, backgroundColor: green, alignItems: 'center', justifyContent: 'center' },
  markText: { color: '#d9ed9c', fontSize: 22, fontWeight: '800' },
  brandName: { color: ink, fontSize: 17, fontWeight: '800', letterSpacing: -0.4 },
  brandCaption: { color: '#7a827d', fontSize: 9, letterSpacing: 0.8, marginTop: 2 },
  bell: { width: 39, height: 39, borderRadius: 14, backgroundColor: '#fff', alignItems: 'center', justifyContent: 'center', borderWidth: 1, borderColor: '#e9ece6' },
  bellText: { color: green, fontSize: 20 },
  eyebrow: { color: green, fontSize: 10, fontWeight: '800', letterSpacing: 1.6 },
  title: { color: ink, fontSize: 31, lineHeight: 39, letterSpacing: -1, fontWeight: '800', marginTop: 7 },
  intro: { color: '#667069', fontSize: 13, marginTop: 8, marginBottom: 20 },
  searchBox: { height: 52, backgroundColor: '#fff', borderWidth: 1, borderColor: '#e9ece6', borderRadius: 16, flexDirection: 'row', alignItems: 'center', paddingHorizontal: 14, gap: 9 },
  searchIcon: { color: green, fontSize: 23 },
  searchInput: { flex: 1, fontSize: 13, color: ink, paddingVertical: 0 },
  clear: { color: green, fontSize: 11, fontWeight: '700' },
  sectionHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'flex-end', marginTop: 29, marginBottom: 13 },
  sectionTitle: { color: ink, fontSize: 18, fontWeight: '800' },
  sectionSub: { color: '#89918b', fontSize: 10, marginTop: 4 },
  count: { color: '#758079', fontSize: 10, paddingBottom: 3 },
  filters: { gap: 8, paddingBottom: 2 },
  filterHeading:{color:'#78817a',fontSize:10,fontWeight:'700',marginTop:12,marginBottom:6},
  filter: { borderRadius: 20, backgroundColor: '#fff', borderColor: '#e5e9e2', borderWidth: 1, paddingHorizontal: 13, paddingVertical: 9 },
  filterActive: { backgroundColor: ink, borderColor: ink },
  filterText: { color: '#68726a', fontSize: 10, fontWeight: '600' },
  filterTextActive: { color: '#fff' },
  emptyCard: { marginTop: 15, padding: 20, minHeight: 245, alignItems: 'center', justifyContent: 'center', backgroundColor: '#fff', borderWidth: 1, borderColor: '#e9ece6', borderRadius: 20 },
  emptyIcon: { width: 55, height: 55, borderRadius: 19, backgroundColor: '#edf3e8', alignItems: 'center', justifyContent: 'center', marginBottom: 13 },
  emptyIconText: { color: green, fontSize: 32 },
  emptyTitle: { color: ink, fontWeight: '800', fontSize: 16 },
  emptyBody: { color: '#78817a', textAlign: 'center', fontSize: 11, lineHeight: 18, marginTop: 8, maxWidth: 285 },
  emptyDivider: { height: 1, backgroundColor: '#edf0ea', alignSelf: 'stretch', marginTop: 17, marginBottom: 12 },
  emptyFoot: { color: '#9aa29b', fontSize: 9, letterSpacing: 0.5 },
  errorIcon: { color: '#b65d4a', fontSize: 25, fontWeight: '800' },
  retry: { backgroundColor: green, borderRadius: 12, paddingHorizontal: 17, paddingVertical: 10, marginTop: 15 },
  retryText: { color: '#fff', fontSize: 11, fontWeight: '700' },
  courseList: { gap: 9, marginTop: 14 },
  loadMore:{alignSelf:'center',backgroundColor:'#fff',borderWidth:1,borderColor:'#e9ece6',borderRadius:12,paddingHorizontal:18,paddingVertical:11,marginTop:14},
  loadMoreText:{color:green,fontSize:11,fontWeight:'700'},
  courseCard: { backgroundColor: '#fff', borderWidth: 1, borderColor: '#e9ece6', borderRadius: 16, padding: 13, flexDirection: 'row', alignItems: 'center', gap: 11 },
  courseMark: { width: 40, height: 40, borderRadius: 13, backgroundColor: '#edf3e8', alignItems: 'center', justifyContent: 'center' },
  courseMarkText: { color: green, fontWeight: '800', fontSize: 16 },
  courseInfo: { flex: 1 },
  courseCode: { color: '#7a827d', fontSize: 9, fontWeight: '700' },
  courseName: { color: ink, fontSize: 12, fontWeight: '700', marginTop: 3 },
  courseMeta: { color: '#828a83', fontSize: 9, marginTop: 4 },
  infoCard: { marginTop: 14, backgroundColor: '#315f4a', borderRadius: 17, padding: 15, flexDirection: 'row', alignItems: 'center', gap: 11 },
  infoDot: { width: 35, height: 35, backgroundColor: '#ffffff1c', borderRadius: 12, alignItems: 'center', justifyContent: 'center' },
  infoDotText: { color: '#d9ed9c', fontSize: 17 },
  infoCopy: { flex: 1 },
  infoTitle: { color: '#fff', fontSize: 12, fontWeight: '700' },
  infoBody: { color: '#d2ded5', fontSize: 9, lineHeight: 14, marginTop: 4 },
  infoArrow: { color: '#d9ed9c', fontSize: 24 },
  footer: { textAlign: 'center', color: '#a0a8a1', fontSize: 9, marginTop: 20 },
  bottomNav: { minHeight: 68, backgroundColor: '#fffffff2', borderTopWidth: 1, borderTopColor: '#e9ece6', flexDirection: 'row', justifyContent: 'space-around', alignItems: 'center', paddingBottom: 6 },
  navItem: { minWidth: 50, alignItems: 'center', gap: 3 },
  navIcon: { color: '#939b94', fontSize: 18, height: 22 },
  navLabel: { color: '#939b94', fontSize: 9 },
  navSelected: { color: green, fontWeight: '800' },
});
