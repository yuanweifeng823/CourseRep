import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { ActivityIndicator, RefreshControl, ScrollView, StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { apiFetch } from '../../api';
import type { Course, CourseFacets, HomeOutput, HomePost } from '../../src/api/types';
import { DepartmentPicker } from '../ui/DepartmentPicker';
import { colors } from '../ui/theme';

type Props = {
  token: string;
  onOpenCourse: (course: Course) => void;
};

export function HomeScreen({ token, onOpenCourse }: Props) {
  const queryClient = useQueryClient();
  const [pickerOpen, setPickerOpen] = useState(false);
  const [saving, setSaving] = useState(false);

  const facets = useQuery({
    queryKey: ['course-facets'],
    queryFn: async () => {
      const response = await apiFetch('/api/v1/course-facets', token);
      if (!response.ok) throw new Error('读取专业列表失败');
      return (await response.json()) as CourseFacets;
    },
  });

  const home = useQuery({
    queryKey: ['home'],
    queryFn: async () => {
      const response = await apiFetch('/api/v1/home', token);
      if (!response.ok) throw new Error('首页暂时无法加载');
      return (await response.json()) as HomeOutput;
    },
  });

  const chooseDepartment = async (department: string) => {
    setSaving(true);
    try {
      const response = await apiFetch('/api/v1/users/me', token, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ department }),
      });
      if (!response.ok) throw new Error('保存专业失败，请稍后再试');
      setPickerOpen(false);
      await queryClient.invalidateQueries({ queryKey: ['home'] });
      await queryClient.invalidateQueries({ queryKey: ['profile'] });
    } finally {
      setSaving(false);
    }
  };

  const data = home.data;
  const needsDepartment = !home.isLoading && (!data || data.needs_department);
  const courses = data?.top_courses ?? [];
  const posts = data?.trending_posts ?? [];

  return (
    <View style={{ flex: 1 }}>
      <ScrollView
        contentContainerStyle={styles.content}
        refreshControl={
          <RefreshControl refreshing={home.isFetching} onRefresh={() => void home.refetch()} tintColor={colors.green} />
        }
      >
        <Text style={styles.eyebrow}>FOR YOU</Text>
        <Text style={styles.title}>{data?.department ? `${data.department} 的今日推荐` : '选一个专业，开始探索'}</Text>
        <Text style={styles.intro}>
          {data?.department
            ? '按你所在专业整理的高分课程与近期热门分享。'
            : '首页会按专业推送课程评分和热门经验帖。'}
        </Text>

        {data?.department ? (
          <TouchableOpacity style={styles.deptChip} onPress={() => setPickerOpen(true)}>
            <Text style={styles.deptChipText}>专业：{data.department}</Text>
            <Text style={styles.deptChipAction}>切换⌄</Text>
          </TouchableOpacity>
        ) : null}

        {home.isLoading ? (
          <View style={styles.loadingCard}>
            <ActivityIndicator color={colors.green} />
            <Text style={styles.muted}>正在读取首页内容…</Text>
          </View>
        ) : home.error ? (
          <View style={styles.loadingCard}>
            <Text style={styles.cardTitle}>首页暂时无法加载</Text>
            <Text style={styles.muted}>请确认后端正在运行，并检查网络。</Text>
            <TouchableOpacity style={styles.primary} onPress={() => void home.refetch()}>
              <Text style={styles.primaryText}>重新加载</Text>
            </TouchableOpacity>
          </View>
        ) : needsDepartment ? (
          <View style={styles.pickCard}>
            <Text style={styles.cardTitle}>先选择你的专业</Text>
            <Text style={styles.muted}>专业只用于首页推荐与课程筛选，可以随时在「我的」里修改。</Text>
            <TouchableOpacity style={styles.primary} onPress={() => setPickerOpen(true)}>
              <Text style={styles.primaryText}>选择专业</Text>
            </TouchableOpacity>
            {saving ? <Text style={styles.muted}>正在保存…</Text> : null}
          </View>
        ) : (
          <>
            <View style={styles.sectionHeader}>
              <Text style={styles.sectionTitle}>近期热门分享</Text>
              <Text style={styles.sectionHint}>按近 30 天浏览</Text>
            </View>
            {posts.length === 0 ? (
              <View style={styles.emptyCard}>
                <Text style={styles.cardTitle}>这个专业还没有热门分享</Text>
                <Text style={styles.muted}>打开课程详情，分享你的修课经验，审核通过后会出现在这里。</Text>
              </View>
            ) : (
              posts.map((post) => (
                <PostCard key={post.id} post={post} onPress={() => onOpenCourse(post.course)} />
              ))
            )}

            <View style={styles.sectionHeader}>
              <Text style={styles.sectionTitle}>评分最高的课程</Text>
              <Text style={styles.sectionHint}>按有效评价平均分</Text>
            </View>
            {courses.length === 0 ? (
              <View style={styles.emptyCard}>
                <Text style={styles.cardTitle}>还没有足够评价</Text>
                <Text style={styles.muted}>课程积累评价后，会按评分排在这里。</Text>
              </View>
            ) : (
              courses.map((course) => (
                <CourseCard key={course.id} course={course} onPress={() => onOpenCourse(course)} />
              ))
            )}
          </>
        )}

        <Text style={styles.footer}>为 CityU 同学整理真实、可信的课程信息</Text>
      </ScrollView>

      <DepartmentPicker
        visible={pickerOpen}
        departments={facets.data?.departments ?? []}
        current={data?.department}
        onSelect={(department) => void chooseDepartment(department)}
        onClose={() => setPickerOpen(false)}
      />
    </View>
  );
}

function PostCard({ post, onPress }: { post: HomePost; onPress: () => void }) {
  return (
    <TouchableOpacity style={styles.card} onPress={onPress}>
      <Text style={styles.cardTitle} numberOfLines={2}>
        {post.title}
      </Text>
      <Text style={styles.meta}>
        {post.course.course_code} · {post.display_name} · {post.attendance_year_label}
      </Text>
      {!!post.body && (
        <Text style={styles.body} numberOfLines={2}>
          {post.body}
        </Text>
      )}
      <Text style={styles.stats}>
        👁 {post.view_count}　♥ {post.like_count}　💬 {post.comment_count}
      </Text>
    </TouchableOpacity>
  );
}

function CourseCard({ course, onPress }: { course: Course; onPress: () => void }) {
  return (
    <TouchableOpacity style={styles.card} onPress={onPress}>
      <View style={styles.courseRow}>
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
        </View>
        <View style={styles.ratingPill}>
          <Text style={styles.ratingValue}>{course.rating.toFixed(1)}</Text>
          <Text style={styles.ratingLabel}>{course.review_count} 评</Text>
        </View>
      </View>
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  content: { paddingHorizontal: 20, paddingBottom: 36 },
  eyebrow: { color: colors.green, fontSize: 10, fontWeight: '800', letterSpacing: 1.6, marginTop: 6 },
  title: { color: colors.ink, fontSize: 27, lineHeight: 34, fontWeight: '800', marginTop: 6, letterSpacing: -0.6 },
  intro: { color: colors.muted, fontSize: 12, lineHeight: 19, marginTop: 8 },
  deptChip: {
    marginTop: 16,
    alignSelf: 'flex-start',
    flexDirection: 'row',
    alignItems: 'center',
    gap: 10,
    backgroundColor: '#fff',
    borderWidth: 1,
    borderColor: colors.line,
    borderRadius: 20,
    paddingHorizontal: 14,
    paddingVertical: 9,
  },
  deptChipText: { color: colors.ink, fontSize: 12, fontWeight: '700' },
  deptChipAction: { color: colors.green, fontSize: 11, fontWeight: '800' },
  sectionHeader: { flexDirection: 'row', alignItems: 'flex-end', justifyContent: 'space-between', marginTop: 26, marginBottom: 10 },
  sectionTitle: { color: colors.ink, fontSize: 17, fontWeight: '800' },
  sectionHint: { color: colors.faint, fontSize: 10 },
  card: {
    backgroundColor: colors.card,
    borderWidth: 1,
    borderColor: colors.line,
    borderRadius: 16,
    padding: 14,
    marginTop: 9,
  },
  cardTitle: { color: colors.ink, fontSize: 14, fontWeight: '800', lineHeight: 20 },
  meta: { color: '#828a83', fontSize: 10, marginTop: 6 },
  body: { color: '#4e5a52', fontSize: 12, lineHeight: 18, marginTop: 7 },
  stats: { color: '#8a938c', fontSize: 10, marginTop: 9 },
  courseRow: { flexDirection: 'row', alignItems: 'center', gap: 11 },
  courseMark: { width: 40, height: 40, borderRadius: 13, backgroundColor: '#edf3e8', alignItems: 'center', justifyContent: 'center' },
  courseMarkText: { color: colors.green, fontWeight: '800', fontSize: 16 },
  code: { color: '#7a827d', fontSize: 9, fontWeight: '700' },
  ratingPill: { alignItems: 'center', backgroundColor: '#edf3e8', borderRadius: 12, paddingHorizontal: 10, paddingVertical: 7 },
  ratingValue: { color: colors.green, fontSize: 16, fontWeight: '800' },
  ratingLabel: { color: '#6f7d72', fontSize: 8, marginTop: 2 },
  loadingCard: {
    marginTop: 18,
    padding: 22,
    alignItems: 'center',
    gap: 8,
    backgroundColor: colors.card,
    borderRadius: 18,
    borderWidth: 1,
    borderColor: colors.line,
  },
  pickCard: {
    marginTop: 18,
    padding: 22,
    backgroundColor: colors.card,
    borderRadius: 18,
    borderWidth: 1,
    borderColor: colors.line,
    gap: 8,
  },
  emptyCard: {
    marginTop: 9,
    padding: 18,
    backgroundColor: colors.card,
    borderRadius: 16,
    borderWidth: 1,
    borderColor: colors.line,
    gap: 6,
  },
  muted: { color: colors.muted, fontSize: 12, lineHeight: 18 },
  primary: {
    marginTop: 8,
    alignSelf: 'flex-start',
    backgroundColor: colors.green,
    borderRadius: 12,
    paddingHorizontal: 18,
    paddingVertical: 11,
  },
  primaryText: { color: '#fff', fontSize: 12, fontWeight: '800' },
  footer: { textAlign: 'center', color: colors.faint, fontSize: 9, marginTop: 26 },
});
