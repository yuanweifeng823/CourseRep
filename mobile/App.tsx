import { StatusBar } from 'expo-status-bar';
import { useCallback, useEffect, useMemo, useState } from 'react';
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

type Course = {
  id: number;
  course_code: string;
  course_name: string;
  professor: string;
  semester: string;
  department: string;
  credits: number;
};

type CourseListResponse = { items: Course[] };

const apiBaseUrl = (process.env.EXPO_PUBLIC_API_URL ||
  (Platform.OS === 'android' ? 'http://10.0.2.2:8080' : 'http://localhost:8080')).replace(/\/$/, '');

export default function App() {
  const [activeTab, setActiveTab] = useState('发现');
  const [query, setQuery] = useState('');
  const [courses, setCourses] = useState<Course[]>([]);
  const [loading, setLoading] = useState(true);
  const [apiError, setApiError] = useState('');
  const loadCourses = useCallback(async (searchTerm: string) => {
    setLoading(true);
    setApiError('');
    try {
      const params = new URLSearchParams({ page: '1', page_size: '100', q: searchTerm });
      const response = await fetch(`${apiBaseUrl}/api/v1/courses?${params.toString()}`, { headers: { Accept: 'application/json' } });
      if (!response.ok) throw new Error(`服务器返回 ${response.status}`);
      const payload = await response.json() as CourseListResponse;
      if (!payload || !Array.isArray(payload.items)) throw new Error('课程接口响应格式不正确');
      setCourses(payload.items);
    } catch {
      setApiError(`暂时无法连接课程服务（${apiBaseUrl}）。请确认后端正在运行，并检查 API 地址。`);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const timer = setTimeout(() => { void loadCourses(query.trim()); }, 300);
    return () => clearTimeout(timer);
  }, [loadCourses, query]);

  const filteredTitle = query.trim();

  return (
    <SafeAreaView style={styles.safe}>
      <StatusBar style="dark" />
      <View style={styles.screen}>
        <ScrollView
          contentContainerStyle={styles.content}
          keyboardShouldPersistTaps="handled"
          refreshControl={<RefreshControl refreshing={loading} onRefresh={() => void loadCourses(query.trim())} tintColor={green} />}
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
              <TouchableOpacity style={styles.retry} onPress={() => void loadCourses(query.trim())}><Text style={styles.retryText}>重新连接</Text></TouchableOpacity>
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
                <View key={course.id} style={styles.courseCard}>
                  <View style={styles.courseMark}><Text style={styles.courseMarkText}>{course.course_code.slice(0, 1)}</Text></View>
                  <View style={styles.courseInfo}>
                    <Text style={styles.courseCode}>{course.course_code} · {course.semester}</Text>
                    <Text style={styles.courseName}>{course.course_name}</Text>
                    <Text style={styles.courseMeta}>{course.professor || '教授待补充'} · {course.department} · {course.credits} 学分</Text>
                  </View>
                </View>
              ))}
            </View>
          )}

          <View style={styles.infoCard}>
            <View style={styles.infoDot}><Text style={styles.infoDotText}>✦</Text></View>
            <View style={styles.infoCopy}>
              <Text style={styles.infoTitle}>分享你的修课经验</Text>
              <Text style={styles.infoBody}>目录开放后，课程评价和经验帖会在对应课程下展示。</Text>
            </View>
            <Text style={styles.infoArrow}>›</Text>
          </View>
          <Text style={styles.footer}>为 CityU 同学整理真实、可信的课程信息</Text>
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
