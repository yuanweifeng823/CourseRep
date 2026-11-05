import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { ActivityIndicator, RefreshControl, ScrollView, StyleSheet, Text, TextInput, TouchableOpacity, View } from 'react-native';
import { apiFetch } from '../../api';
import type { Course, CourseFacets, CourseListResponse } from '../../src/api/types';
import { DepartmentPicker } from '../ui/DepartmentPicker';
import { colors } from '../ui/theme';

type SortKey = 'course' | 'rating' | 'reviews' | 'posts';

const sorts: { key: SortKey; label: string }[] = [
  { key: 'course', label: '课程代码' },
  { key: 'rating', label: '评分' },
  { key: 'reviews', label: '评价数' },
  { key: 'posts', label: '分享数' },
];

type Props = {
  token: string;
  onOpenCourse: (course: Course) => void;
};

export function CourseCatalog({ token, onOpenCourse }: Props) {
  const [query, setQuery] = useState('');
  const [debounced, setDebounced] = useState('');
  const [department, setDepartment] = useState('');
  const [sort, setSort] = useState<SortKey>('course');
  const [pickerOpen, setPickerOpen] = useState(false);

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(query.trim()), 300);
    return () => clearTimeout(timer);
  }, [query]);

  const facets = useQuery({
    queryKey: ['course-facets'],
    queryFn: async () => {
      const response = await apiFetch('/api/v1/course-facets', token);
      if (!response.ok) throw new Error('读取专业列表失败');
      return (await response.json()) as CourseFacets;
    },
  });

  const catalog = useInfiniteQuery({
    queryKey: ['catalog', debounced, department, sort],
    initialPageParam: 1,
    queryFn: async ({ pageParam }) => {
      const params = new URLSearchParams({ page: String(pageParam), page_size: '50', sort });
      if (debounced) params.set('q', debounced);
      if (department) params.set('department', department);
      const response = await apiFetch(`/api/v1/courses?${params.toString()}`, token);
      if (!response.ok) throw new Error('课程列表暂时无法加载');
      return (await response.json()) as CourseListResponse;
    },
    getNextPageParam: (lastPage) => {
      const count = lastPage.items?.length ?? 0;
      return count > 0 && count === (lastPage.page_size ?? 50) ? (lastPage.page ?? 1) + 1 : undefined;
    },
  });

  const courses = (catalog.data?.pages.flatMap((page) => page.items ?? []) ?? []).filter(
    (item): item is NonNullable<typeof item> => item !== null,
  );

  return (
    <View style={{ flex: 1 }}>
      <ScrollView
        contentContainerStyle={styles.content}
        keyboardShouldPersistTaps="handled"
        refreshControl={
          <RefreshControl refreshing={catalog.isFetching} onRefresh={() => void catalog.refetch()} tintColor={colors.green} />
        }
      >
        <Text style={styles.eyebrow}>COURSE CATALOG</Text>
        <Text style={styles.title}>全部课程</Text>
        <Text style={styles.intro}>按课程代码、名称或教授搜索，也可以只看某个专业。</Text>

        <View style={styles.searchBox}>
          <Text style={styles.searchIcon}>⌕</Text>
          <TextInput
            value={query}
            onChangeText={setQuery}
            placeholder="课程代码、课程名称或教授"
            placeholderTextColor="#929a93"
            style={styles.searchInput}
            returnKeyType="search"
            accessibilityLabel="搜索课程"
          />
          {query.length > 0 ? (
            <TouchableOpacity onPress={() => setQuery('')}>
              <Text style={styles.clear}>清除</Text>
            </TouchableOpacity>
          ) : null}
        </View>

        <TouchableOpacity style={styles.deptSelect} onPress={() => setPickerOpen(true)}>
          <Text style={styles.deptLabel}>专业</Text>
          <Text style={styles.deptValue}>{department || '全部专业'}</Text>
          {department ? (
            <TouchableOpacity onPress={() => setDepartment('')} hitSlop={10}>
              <Text style={styles.clear}>全部</Text>
            </TouchableOpacity>
          ) : null}
          <Text style={styles.deptArrow}>⌄</Text>
        </TouchableOpacity>

        <Text style={styles.filterHeading}>排序</Text>
        <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.filters}>
          {sorts.map((item) => (
            <TouchableOpacity
              key={item.key}
              onPress={() => setSort(item.key)}
              style={[styles.filter, sort === item.key && styles.filterActive]}
            >
              <Text style={[styles.filterText, sort === item.key && styles.filterTextActive]}>{item.label}</Text>
            </TouchableOpacity>
          ))}
        </ScrollView>

        <View style={styles.sectionHeader}>
          <Text style={styles.sectionTitle}>{department || '全部专业'}</Text>
          <Text style={styles.count}>{courses.length} 门</Text>
        </View>

        {catalog.isLoading ? (
          <View style={styles.emptyCard}>
            <ActivityIndicator color={colors.green} />
            <Text style={styles.muted}>正在读取课程目录…</Text>
          </View>
        ) : catalog.error ? (
          <View style={styles.emptyCard}>
            <Text style={styles.cardTitle}>课程服务暂不可用</Text>
            <Text style={styles.muted}>请确认后端正在运行，并检查手机与电脑是否在同一网络。</Text>
            <TouchableOpacity style={styles.primary} onPress={() => void catalog.refetch()}>
              <Text style={styles.primaryText}>重新加载</Text>
            </TouchableOpacity>
          </View>
        ) : courses.length === 0 ? (
          <View style={styles.emptyCard}>
            <Text style={styles.cardTitle}>{debounced ? '没有匹配的课程' : '这个专业还没有课程'}</Text>
            <Text style={styles.muted}>
              {debounced ? `没有找到“${debounced}”，换个关键词试试。` : '课程导入后会出现在这里。'}
            </Text>
          </View>
        ) : (
          courses.map((course) => (
            <TouchableOpacity key={course.id} style={styles.card} onPress={() => onOpenCourse(course)}>
              <View style={styles.courseMark}>
                <Text style={styles.courseMarkText}>{course.course_code.slice(0, 1)}</Text>
              </View>
              <View style={{ flex: 1 }}>
                <Text style={styles.code}>
                  {course.course_code} · {course.semester}
                </Text>
                <Text style={styles.cardTitle} numberOfLines={2}>
                  {course.course_name}
                </Text>
                <Text style={styles.meta}>
                  {course.professor || '教授待补充'} · {course.department} · {course.credits} 学分
                </Text>
                <Text style={styles.meta}>
                  评分 {course.rating.toFixed(1)} · {course.review_count} 条评价 · {course.post_count} 篇分享
                </Text>
              </View>
            </TouchableOpacity>
          ))
        )}

        {catalog.hasNextPage ? (
          <TouchableOpacity
            disabled={catalog.isFetchingNextPage}
            style={styles.loadMore}
            onPress={() => void catalog.fetchNextPage()}
          >
            <Text style={styles.loadMoreText}>{catalog.isFetchingNextPage ? '正在加载…' : '加载更多课程'}</Text>
          </TouchableOpacity>
        ) : null}
      </ScrollView>

      <DepartmentPicker
        visible={pickerOpen}
        departments={facets.data?.departments ?? []}
        current={department}
        title="按专业筛选"
        hint="选择「全部专业」可以查看所有课程。"
        onSelect={(value) => {
          setDepartment(value);
          setPickerOpen(false);
        }}
        onClose={() => setPickerOpen(false)}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  content: { paddingHorizontal: 20, paddingBottom: 40 },
  eyebrow: { color: colors.green, fontSize: 10, fontWeight: '800', letterSpacing: 1.6, marginTop: 6 },
  title: { color: colors.ink, fontSize: 27, fontWeight: '800', marginTop: 6, letterSpacing: -0.6 },
  intro: { color: colors.muted, fontSize: 12, lineHeight: 19, marginTop: 8 },
  searchBox: {
    height: 50,
    backgroundColor: '#fff',
    borderWidth: 1,
    borderColor: colors.line,
    borderRadius: 15,
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 14,
    gap: 9,
    marginTop: 16,
  },
  searchIcon: { color: colors.green, fontSize: 21 },
  searchInput: { flex: 1, fontSize: 13, color: colors.ink, paddingVertical: 0 },
  clear: { color: colors.green, fontSize: 11, fontWeight: '700' },
  deptSelect: {
    marginTop: 10,
    height: 48,
    backgroundColor: '#fff',
    borderWidth: 1,
    borderColor: colors.line,
    borderRadius: 15,
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 14,
    gap: 12,
  },
  deptLabel: { color: colors.faint, fontSize: 10, fontWeight: '700' },
  deptValue: { flex: 1, color: colors.ink, fontSize: 13, fontWeight: '700' },
  deptArrow: { color: colors.green, fontSize: 15 },
  filterHeading: { color: colors.muted, fontSize: 10, fontWeight: '700', marginTop: 14, marginBottom: 7 },
  filters: { gap: 8 },
  filter: {
    borderRadius: 20,
    backgroundColor: '#fff',
    borderColor: '#e5e9e2',
    borderWidth: 1,
    paddingHorizontal: 13,
    paddingVertical: 8,
  },
  filterActive: { backgroundColor: colors.ink, borderColor: colors.ink },
  filterText: { color: '#68726a', fontSize: 10, fontWeight: '600' },
  filterTextActive: { color: '#fff' },
  sectionHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'flex-end', marginTop: 22, marginBottom: 4 },
  sectionTitle: { color: colors.ink, fontSize: 17, fontWeight: '800' },
  count: { color: colors.faint, fontSize: 10, paddingBottom: 3 },
  card: {
    backgroundColor: '#fff',
    borderWidth: 1,
    borderColor: colors.line,
    borderRadius: 16,
    padding: 13,
    flexDirection: 'row',
    alignItems: 'center',
    gap: 11,
    marginTop: 9,
  },
  courseMark: { width: 40, height: 40, borderRadius: 13, backgroundColor: '#edf3e8', alignItems: 'center', justifyContent: 'center' },
  courseMarkText: { color: colors.green, fontWeight: '800', fontSize: 16 },
  code: { color: '#7a827d', fontSize: 9, fontWeight: '700' },
  cardTitle: { color: colors.ink, fontSize: 13, fontWeight: '700', marginTop: 3, lineHeight: 18 },
  meta: { color: '#828a83', fontSize: 9, marginTop: 4 },
  muted: { color: colors.muted, fontSize: 12, lineHeight: 18 },
  emptyCard: {
    marginTop: 14,
    padding: 20,
    alignItems: 'center',
    gap: 8,
    backgroundColor: '#fff',
    borderRadius: 18,
    borderWidth: 1,
    borderColor: colors.line,
  },
  primary: { marginTop: 8, backgroundColor: colors.green, borderRadius: 12, paddingHorizontal: 18, paddingVertical: 11 },
  primaryText: { color: '#fff', fontSize: 12, fontWeight: '800' },
  loadMore: {
    alignSelf: 'center',
    backgroundColor: '#fff',
    borderWidth: 1,
    borderColor: colors.line,
    borderRadius: 12,
    paddingHorizontal: 18,
    paddingVertical: 11,
    marginTop: 14,
  },
  loadMoreText: { color: colors.green, fontSize: 11, fontWeight: '700' },
  clearDept: { alignSelf: 'center', paddingVertical: 8 },
  clearDeptText: { color: colors.muted, fontSize: 11 },
});
