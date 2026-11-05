import { useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Image,
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  TouchableOpacity,
  View,
} from 'react-native';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { apiBaseUrl, apiFetch } from '../../api';
import type { Comment, CommentListResponse, Course, FeedPost, PostDetailPost, PostDetailResponse } from '../api/types';
import { colors } from '../ui/theme';

type Props = {
  token: string;
  postId: number;
  course: Course;
  seed?: Partial<FeedPost>;
  onBack: () => void;
  onOpenCourse: (course: Course) => void;
};

export function PostDetailScreen({ token, postId, course, seed, onBack, onOpenCourse }: Props) {
  const queryClient = useQueryClient();
  const [liked, setLiked] = useState(seed?.liked_by_me ?? false);
  const [likeCount, setLikeCount] = useState(seed?.like_count ?? 0);
  const [bookmarked, setBookmarked] = useState(seed?.bookmarked_by_me ?? false);
  const [commentCount, setCommentCount] = useState(seed?.comment_count ?? 0);
  const [comment, setComment] = useState('');
  const [sending, setSending] = useState(false);

  const detail = useQuery({
    queryKey: ['post-detail', postId],
    queryFn: async () => {
      const response = await apiFetch(`/api/v1/posts/${postId}`, token);
      if (!response.ok) throw new Error('经验帖暂时无法查看');
      return (await response.json()) as PostDetailResponse;
    },
  });

  const comments = useQuery({
    queryKey: ['post-comments', postId],
    queryFn: async () => {
      const response = await apiFetch(`/api/v1/posts/${postId}/comments`, token);
      if (!response.ok) throw new Error('评论暂时无法加载');
      return (await response.json()) as CommentListResponse;
    },
  });

  const post: PostDetailPost = detail.data?.post ?? {
    id: postId,
    course_id: course.id,
    title: seed?.title ?? '',
    body: seed?.body ?? '',
    image_urls: seed?.image_urls ?? [],
    topic_tags: seed?.topic_tags ?? [],
    attendance_year: seed?.attendance_year ?? null,
    attendance_year_label: seed?.attendance_year_label ?? '年份未注明',
    display_identity: seed?.display_identity ?? 'anonymous',
    display_name: seed?.display_name ?? '同校已验证',
    created_at: String(seed?.created_at ?? ''),
  };

  const toggleLike = async () => {
    const next = !liked;
    setLiked(next);
    setLikeCount((value) => Math.max(0, value + (next ? 1 : -1)));
    const response = await apiFetch(`/api/v1/posts/${postId}/like`, token, { method: next ? 'PUT' : 'DELETE' });
    if (!response.ok) {
      setLiked(!next);
      setLikeCount((value) => Math.max(0, value + (next ? -1 : 1)));
      Alert.alert('操作失败', '请稍后再试');
    }
  };

  const toggleBookmark = async () => {
    const next = !bookmarked;
    setBookmarked(next);
    const response = await apiFetch(`/api/v1/posts/${postId}/bookmark`, token, { method: next ? 'PUT' : 'DELETE' });
    if (!response.ok) {
      setBookmarked(!next);
      Alert.alert('操作失败', '请稍后再试');
    }
  };

  const submitComment = async () => {
    const body = comment.trim();
    if (!body) return;
    setSending(true);
    try {
      const response = await apiFetch(`/api/v1/posts/${postId}/comments`, token, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ body }),
      });
      const data = (await response.json()) as { error?: string };
      if (!response.ok) throw new Error(data.error || '评论失败');
      setComment('');
      setCommentCount((value) => value + 1);
      await queryClient.invalidateQueries({ queryKey: ['post-comments', postId] });
      await queryClient.invalidateQueries({ queryKey: ['feed'] });
    } catch (error) {
      Alert.alert('无法评论', error instanceof Error ? error.message : '请稍后再试');
    } finally {
      setSending(false);
    }
  };

  const commentItems: Comment[] = comments.data?.items ?? [];

  return (
    <KeyboardAvoidingView style={styles.container} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
      <View style={styles.header}>
        <TouchableOpacity onPress={onBack} accessibilityLabel="返回">
          <Text style={styles.back}>‹</Text>
        </TouchableOpacity>
        <Text style={styles.headerTitle}>经验帖</Text>
        <View style={{ width: 24 }} />
      </View>

      <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
        <Text style={styles.title}>{post.title}</Text>
        <Text style={styles.meta}>
          {post.display_name} · {post.created_at.slice(0, 10)} · 修读年份：{post.attendance_year_label}
        </Text>

        {(post.image_urls ?? []).map((uri) => (
          <Image
            key={uri}
            source={{ uri: `${apiBaseUrl}${uri}`, headers: { Authorization: `Bearer ${token}` } }}
            style={styles.image}
            resizeMode="cover"
          />
        ))}

        <Text style={styles.body}>{post.body}</Text>

        {(post.topic_tags ?? []).length > 0 && (
          <View style={styles.tagRow}>
            {(post.topic_tags ?? []).map((tag) => (
              <View key={tag} style={styles.tag}>
                <Text style={styles.tagText}>{tag}</Text>
              </View>
            ))}
          </View>
        )}

        <TouchableOpacity style={styles.courseCard} onPress={() => onOpenCourse(course)} activeOpacity={0.85}>
          <View style={{ flex: 1 }}>
            <Text style={styles.courseLabel}>关联课程</Text>
            <Text style={styles.courseCode}>{course.course_code}</Text>
            <Text style={styles.courseName} numberOfLines={1}>
              {course.course_name}
            </Text>
            <Text style={styles.courseMeta}>
              {course.department} · {course.semester}
            </Text>
          </View>
          <Text style={styles.courseChevron}>›</Text>
        </TouchableOpacity>

        <View style={styles.actionRow}>
          <TouchableOpacity style={[styles.action, liked && styles.actionActive]} onPress={() => void toggleLike()}>
            <Text style={[styles.actionIcon, liked && styles.actionTextActive]}>{liked ? '♥' : '♡'}</Text>
            <Text style={[styles.actionText, liked && styles.actionTextActive]}>{likeCount}</Text>
          </TouchableOpacity>
          <TouchableOpacity style={[styles.action, bookmarked && styles.actionActive]} onPress={() => void toggleBookmark()}>
            <Text style={[styles.actionIcon, bookmarked && styles.actionTextActive]}>{bookmarked ? '★' : '☆'}</Text>
            <Text style={[styles.actionText, bookmarked && styles.actionTextActive]}>收藏</Text>
          </TouchableOpacity>
          <View style={styles.action}>
            <Text style={styles.actionIcon}>💬</Text>
            <Text style={styles.actionText}>{commentCount}</Text>
          </View>
        </View>

        <Text style={styles.sectionTitle}>评论 {commentCount}</Text>
        {comments.isLoading ? (
          <ActivityIndicator color={colors.green} style={{ marginVertical: 16 }} />
        ) : comments.isError ? (
          <Text style={styles.emptyText}>评论暂时无法加载</Text>
        ) : commentItems.length === 0 ? (
          <Text style={styles.emptyText}>还没有评论，来说两句。</Text>
        ) : (
          commentItems.map((entry) => (
            <View key={entry.id} style={styles.commentRow}>
              <View style={styles.avatar}>
                <Text style={styles.avatarText}>{entry.author.slice(0, 1)}</Text>
              </View>
              <View style={{ flex: 1 }}>
                <Text style={styles.commentMeta}>
                  {entry.author}
                  {entry.is_post_author ? ' · 作者' : ''} · {entry.created_at.slice(0, 10)}
                </Text>
                <Text style={styles.commentBody}>{entry.body}</Text>
              </View>
            </View>
          ))
        )}

        {detail.isError ? <Text style={styles.emptyText}>内容可能已下架或需要重新审核。</Text> : null}
      </ScrollView>

      <View style={styles.composerRow}>
        <TextInput
          value={comment}
          onChangeText={setComment}
          placeholder="友善评论，仅一层回复"
          placeholderTextColor="#9aa39c"
          style={styles.composerInput}
          maxLength={500}
          multiline
        />
        <TouchableOpacity style={[styles.sendButton, (!comment.trim() || sending) && styles.sendDisabled]} onPress={() => void submitComment()} disabled={!comment.trim() || sending}>
          <Text style={styles.sendText}>{sending ? '…' : '发送'}</Text>
        </TouchableOpacity>
      </View>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: colors.paper },
  header: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', paddingHorizontal: 16, paddingVertical: 12 },
  back: { color: colors.ink, fontSize: 26, paddingHorizontal: 4 },
  headerTitle: { color: colors.ink, fontSize: 16, fontWeight: '800' },
  content: { paddingHorizontal: 18, paddingBottom: 28 },
  title: { color: colors.ink, fontSize: 21, fontWeight: '800', lineHeight: 28, letterSpacing: -0.3 },
  meta: { color: colors.muted, fontSize: 11, marginTop: 8 },
  image: { width: '100%', height: 220, borderRadius: 14, marginTop: 12, backgroundColor: '#e9ece4' },
  body: { color: '#3c463f', fontSize: 14, lineHeight: 23, marginTop: 14 },
  tagRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 7, marginTop: 14 },
  tag: { backgroundColor: '#e7ece1', borderRadius: 999, paddingHorizontal: 10, paddingVertical: 5 },
  tagText: { color: colors.green, fontSize: 10, fontWeight: '700' },
  courseCard: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#fff',
    borderRadius: 14,
    borderWidth: 1,
    borderColor: colors.line,
    padding: 13,
    marginTop: 16,
  },
  courseLabel: { color: colors.muted, fontSize: 9, letterSpacing: 0.6 },
  courseCode: { color: colors.green, fontSize: 13, fontWeight: '800', marginTop: 3 },
  courseName: { color: colors.ink, fontSize: 13, fontWeight: '700', marginTop: 2 },
  courseMeta: { color: colors.muted, fontSize: 10, marginTop: 3 },
  courseChevron: { color: colors.faint, fontSize: 22 },
  actionRow: { flexDirection: 'row', gap: 9, marginTop: 16 },
  action: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 5,
    backgroundColor: '#fff',
    borderRadius: 999,
    borderWidth: 1,
    borderColor: colors.line,
    paddingHorizontal: 14,
    paddingVertical: 8,
  },
  actionActive: { backgroundColor: colors.green, borderColor: colors.green },
  actionIcon: { color: '#8d968f', fontSize: 14 },
  actionText: { color: '#8d968f', fontSize: 11, fontWeight: '700' },
  actionTextActive: { color: '#fff' },
  sectionTitle: { color: colors.ink, fontSize: 13, fontWeight: '800', marginTop: 22, marginBottom: 6 },
  emptyText: { color: colors.muted, fontSize: 11, marginTop: 8 },
  commentRow: { flexDirection: 'row', gap: 9, marginTop: 12 },
  avatar: { width: 28, height: 28, borderRadius: 999, backgroundColor: '#e7ece1', alignItems: 'center', justifyContent: 'center' },
  avatarText: { color: colors.green, fontSize: 12, fontWeight: '800' },
  commentMeta: { color: '#8d968f', fontSize: 10 },
  commentBody: { color: '#3c463f', fontSize: 12, lineHeight: 18, marginTop: 3 },
  composerRow: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    gap: 8,
    paddingHorizontal: 14,
    paddingVertical: 10,
    borderTopWidth: 1,
    borderTopColor: colors.line,
    backgroundColor: colors.paper,
  },
  composerInput: {
    flex: 1,
    maxHeight: 90,
    backgroundColor: '#fff',
    borderWidth: 1,
    borderColor: colors.line,
    borderRadius: 12,
    paddingHorizontal: 12,
    paddingVertical: 9,
    color: colors.ink,
    fontSize: 12,
  },
  sendButton: { backgroundColor: colors.green, borderRadius: 11, paddingHorizontal: 16, paddingVertical: 10 },
  sendDisabled: { opacity: 0.5 },
  sendText: { color: '#fff', fontSize: 12, fontWeight: '800' },
});
