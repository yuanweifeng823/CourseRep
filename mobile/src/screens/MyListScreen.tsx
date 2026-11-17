import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ActivityIndicator, ScrollView, StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { apiFetch } from '../../api';
import type {
  Bookmark,
  BookmarkListResponse,
  CompletedCourse,
  CompletedCourseListResponse,
  OwnPost,
  OwnPostListResponse,
  OwnReview,
  OwnReviewListResponse,
} from '../../src/api/types';
import { colors, statusLabel } from '../ui/theme';

export type MyListKind = 'courses' | 'reviews' | 'posts' | 'bookmarks';

export type EditTarget = { kind: 'review'; item: OwnReview } | { kind: 'post'; item: OwnPost };

const titles: Record<MyListKind, string> = {
  courses: '已修读课程',
  reviews: '我的评价',
  posts: '我的经验帖',
  bookmarks: '我的收藏',
};

type Props = {
  token: string;
  kind: MyListKind;
  focusId?: number;
  onBack: () => void;
  onEdit: (target: EditTarget) => void;
  onOpenBookmark: (item: Bookmark) => void;
};

export function MyListScreen({ token, kind, focusId, onBack, onEdit, onOpenBookmark }: Props) {
  const [showAll, setShowAll] = useState(false);
  const selectedId = showAll ? undefined : focusId;
  const suffix = selectedId === undefined ? '' : `?id=${selectedId}`;
  const courses = useQuery({
    queryKey: ['my-courses'],
    enabled: kind === 'courses',
    queryFn: async () => {
      const response = await apiFetch('/api/v1/users/me/courses', token);
      if (!response.ok) throw new Error();
      return ((await response.json()) as CompletedCourseListResponse).items ?? [];
    },
  });
  const reviews = useQuery({
    queryKey: selectedId === undefined ? ['my-reviews'] : ['my-reviews', selectedId],
    enabled: kind === 'reviews',
    queryFn: async () => {
      const response = await apiFetch(`/api/v1/users/me/reviews${suffix}`, token);
      if (!response.ok) throw new Error();
      return ((await response.json()) as OwnReviewListResponse).items ?? [];
    },
  });
  const posts = useQuery({
    queryKey: selectedId === undefined ? ['my-posts'] : ['my-posts', selectedId],
    enabled: kind === 'posts',
    queryFn: async () => {
      const response = await apiFetch(`/api/v1/users/me/posts${suffix}`, token);
      if (!response.ok) throw new Error();
      return ((await response.json()) as OwnPostListResponse).items ?? [];
    },
  });
  const bookmarks = useQuery({
    queryKey: ['my-bookmarks'],
    enabled: kind === 'bookmarks',
    queryFn: async () => {
      const response = await apiFetch('/api/v1/users/me/bookmarks', token);
      if (!response.ok) throw new Error();
      return ((await response.json()) as BookmarkListResponse).items ?? [];
    },
  });

  const currentQuery = { courses, reviews, posts, bookmarks }[kind];
  const loading = currentQuery.isPending;
  const failed = currentQuery.isError;
  const showFocused = selectedId !== undefined;
  const reviewItems = reviews.data?.filter((item) => !showFocused || item.id === focusId) ?? [];
  const postItems = posts.data?.filter((item) => !showFocused || item.id === focusId) ?? [];

  return (
    <View style={styles.screen}>
      <View style={styles.header}>
        <TouchableOpacity onPress={onBack} hitSlop={10}>
          <Text style={styles.back}>‹ 返回</Text>
        </TouchableOpacity>
        <Text style={styles.heading}>{titles[kind]}</Text>
      </View>
      <ScrollView contentContainerStyle={styles.content}>
        {loading ? <ActivityIndicator color={colors.green} style={{ marginTop: 30 }} /> : null}
        {failed && <TouchableOpacity onPress={() => void currentQuery.refetch()}>
          <Text style={styles.empty}>暂时无法加载，点击重试</Text>
        </TouchableOpacity>}
        {showFocused && !loading && !failed && <TouchableOpacity onPress={() => setShowAll(true)}>
          <Text style={styles.link}>查看全部{kind === 'reviews' ? '评价' : '经验帖'}</Text>
        </TouchableOpacity>}
        {!loading && !failed && kind === 'courses' &&
          (courses.data?.length ? (
            courses.data.map((item: CompletedCourse) => (
              <View key={item.id} style={styles.card}>
                <Text style={styles.title}>
                  {item.course_code} · {item.course_name}
                </Text>
                <Text style={styles.meta}>
                  {item.semester} · {item.department}
                </Text>
              </View>
            ))
          ) : (
            <Text style={styles.empty}>还没有标记已修读课程，可在课程详情页添加</Text>
          ))}
        {!loading && !failed && kind === 'reviews' &&
          (reviewItems.length ? (
            reviewItems.map((item: OwnReview) => (
              <View key={item.id} style={styles.card}>
                <Text style={styles.title}>
                  {item.course_code} · {item.course_name}
                </Text>
                <Text style={styles.meta}>
                  {statusLabel(item.status)} · {item.semester}
                </Text>
                <Text style={styles.body}>
                  给分 {item.grading} · 工作量 {item.workload} · 考核 {item.assessment} · 实用性 {item.usefulness}
                </Text>
                {!!item.body && <Text style={styles.body}>{item.body}</Text>}
                {['published', 'rejected'].includes(item.status) && (
                  <TouchableOpacity onPress={() => onEdit({ kind: 'review', item })}>
                    <Text style={styles.link}>编辑并重新提交审核</Text>
                  </TouchableOpacity>
                )}
              </View>
            ))
          ) : (
            <Text style={styles.empty}>{showFocused ? '暂时无法找到这条评价' : '还没有提交过评价'}</Text>
          ))}
        {!loading && !failed && kind === 'posts' &&
          (postItems.length ? (
            postItems.map((item: OwnPost) => (
              <View key={item.id} style={styles.card}>
                <Text style={styles.title}>{item.title}</Text>
                <Text style={styles.meta}>
                  {item.course_code} · {item.attendance_year_label} · {statusLabel(item.status)}
                </Text>
                <Text style={styles.body} numberOfLines={3}>
                  {item.body}
                </Text>
                {['published', 'rejected'].includes(item.status) && (
                  <TouchableOpacity onPress={() => onEdit({ kind: 'post', item })}>
                    <Text style={styles.link}>编辑并重新提交审核</Text>
                  </TouchableOpacity>
                )}
              </View>
            ))
          ) : (
            <Text style={styles.empty}>{showFocused ? '暂时无法找到这篇经验帖' : '还没有发布过经验帖'}</Text>
          ))}
        {!loading && !failed && kind === 'bookmarks' &&
          (bookmarks.data?.length ? (
            bookmarks.data.map((item: Bookmark) => (
              <TouchableOpacity
                key={item.id}
                style={styles.card}
                accessibilityRole="button"
                onPress={() => onOpenBookmark(item)}
              >
                <Text style={styles.title}>{item.title}</Text>
                <Text style={styles.meta}>
                  {item.course_code} · {item.attendance_year_label}
                </Text>
                <Text style={styles.body} numberOfLines={2}>
                  {item.body}
                </Text>
              </TouchableOpacity>
            ))
          ) : (
            <Text style={styles.empty}>还没有收藏经验帖</Text>
          ))}
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1 },
  header: {
    height: 52,
    backgroundColor: '#fff',
    borderBottomWidth: 1,
    borderBottomColor: colors.line,
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 20,
  },
  back: { color: colors.green, fontWeight: '700', fontSize: 14 },
  heading: { marginLeft: 18, color: colors.ink, fontSize: 17, fontWeight: '800' },
  content: { padding: 18, paddingBottom: 40 },
  card: {
    backgroundColor: '#fff',
    borderWidth: 1,
    borderColor: colors.line,
    borderRadius: 15,
    padding: 14,
    marginBottom: 10,
  },
  title: { color: colors.ink, fontSize: 14, fontWeight: '800', lineHeight: 20 },
  meta: { color: '#839087', fontSize: 11, marginTop: 6 },
  body: { color: '#4e5a52', fontSize: 12, lineHeight: 19, marginTop: 7 },
  link: { color: colors.green, fontSize: 11, fontWeight: '800', marginTop: 9 },
  empty: { textAlign: 'center', color: '#77827a', marginTop: 30, fontSize: 12 },
});
