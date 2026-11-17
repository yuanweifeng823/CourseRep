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
import * as ImagePicker from 'expo-image-picker';
import { apiBaseUrl, apiFetch } from '../../api';
import type { CommunitySettings, CourseDetailCourse, TopicTagListResponse } from '../api/types';
import { colors } from '../ui/theme';
import { CoursePicker } from '../ui/CoursePicker';
import { AttendanceYearPicker } from '../ui/AttendanceYearPicker';
import { useUnsavedChanges, type LeaveState } from '../ui/unsavedChanges';

type SelectedCourse = Pick<CourseDetailCourse, 'id' | 'course_code' | 'course_name'>;

type Props = {
  token: string;
  initialCourse?: SelectedCourse;
  onDirtyChange: (dirty: LeaveState) => void;
  onClose: () => void;
  onPublished: () => void;
};

const MAX_IMAGES = 9;
const MAX_BODY = 2000;

export function ComposePostScreen({ token, initialCourse, onClose, onPublished, onDirtyChange }: Props) {
  const queryClient = useQueryClient();
  const [course, setCourse] = useState<SelectedCourse | null>(initialCourse ?? null);
  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  const [year, setYear] = useState('unspecified');
  const [yearPicker, setYearPicker] = useState(false);
  const [anonymous, setAnonymous] = useState(false);
  const [images, setImages] = useState<string[]>([]);
  const [selectedTags, setSelectedTags] = useState<string[]>([]);
  const [operation, setOperation] = useState<'images' | 'submit' | null>(null);
  const busy = operation !== null;

  useUnsavedChanges(busy ? 'busy' : !!title || !!body || year !== 'unspecified' || anonymous ||
    images.length > 0 || selectedTags.length > 0, onDirtyChange);

  const policy = useQuery({
    queryKey: ['community-guidelines'],
    enabled: course !== null,
    queryFn: async () => {
      const response = await apiFetch('/api/v1/community-guidelines', token);
      if (!response.ok) throw new Error('社区信息暂不可用');
      return (await response.json()) as CommunitySettings;
    },
  });

  const topics = useQuery({
    queryKey: ['topic-tags'],
    enabled: course !== null,
    queryFn: async () => {
      const response = await apiFetch('/api/v1/topic-tags', token);
      if (!response.ok) throw new Error('读取话题失败');
      return (await response.json()) as TopicTagListResponse;
    },
  });

  const pickImages = async () => {
    if (busy || images.length >= MAX_IMAGES) return;
    setOperation('images');
    try {
      const result = await ImagePicker.launchImageLibraryAsync({
        mediaTypes: ['images'],
        allowsMultipleSelection: true,
        selectionLimit: MAX_IMAGES - images.length,
        quality: 0.9,
      });
      if (result.canceled) return;
      for (const asset of result.assets.slice(0, MAX_IMAGES - images.length)) {
        const form = new FormData();
        form.append('image', {
          uri: asset.uri,
          name: asset.fileName || 'photo.jpg',
          type: asset.mimeType || 'image/jpeg',
        } as unknown as Blob);
        const response = await apiFetch('/api/v1/uploads', token, { method: 'POST', body: form });
        const data = (await response.json()) as { image_url?: string; error?: string };
        if (!response.ok || !data.image_url) throw new Error(data.error || '图片上传失败');
        const image = data.image_url;
        setImages((value) => value.includes(image) ? value : [...value, image]);
      }
    } catch (error) {
      Alert.alert('图片上传失败', error instanceof Error ? error.message : '请稍后再试');
    } finally {
      setOperation(null);
    }
  };

  const toggleTag = (name: string) => {
    setSelectedTags((value) => (value.includes(name) ? value.filter((item) => item !== name) : [...value, name]));
  };

  const submit = async () => {
    if (!course || busy) return;
    const yearValue = year === 'unspecified' ? null : Number(year);
    if (!title.trim()) return Alert.alert('需要标题', '请填写经验帖标题。');
    if (!body.trim()) return Alert.alert('需要正文', '请填写经验帖正文。');
    setOperation('submit');
    try {
      const response = await apiFetch(`/api/v1/courses/${course.id}/posts`, token, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title: title.trim(),
          body: body.trim(),
          image_urls: images,
          topic_tags: selectedTags,
          attendance_year: yearValue,
          display_identity: anonymous ? 'anonymous' : 'named',
        }),
      });
      const data = (await response.json()) as { error?: string };
      if (!response.ok) throw new Error(data.error || '提交失败');
      void queryClient.invalidateQueries({ queryKey: ['my-posts'] });
      onPublished();
      Alert.alert('已提交审核', '审核通过后会在课程和分享页公开，可在「我的经验帖」查看状态。');
    } catch (error) {
      Alert.alert('无法发布', error instanceof Error ? error.message : '请稍后再试');
    } finally {
      setOperation(null);
    }
  };

  if (!course) {
    return <CoursePicker token={token} onClose={onClose} onSelect={setCourse} />;
  }

  return (
    <KeyboardAvoidingView style={styles.container} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
      <View style={styles.header}>
        <TouchableOpacity onPress={onClose} hitSlop={12} accessibilityRole="button" accessibilityLabel="取消">
          <Text style={styles.back}>‹</Text>
        </TouchableOpacity>
        <Text style={styles.headerTitle}>写经验帖</Text>
        <View style={{ width: 24 }} />
      </View>
      <ScrollView contentContainerStyle={{ paddingBottom: 40 }} keyboardShouldPersistTaps="handled">
        <View style={styles.courseCard}>
          <View style={{ flex: 1 }}>
            <Text style={styles.courseCode}>{course.course_code}</Text>
            <Text style={styles.courseName} numberOfLines={1}>
              {course.course_name}
            </Text>
          </View>
          <TouchableOpacity disabled={busy} onPress={() => setCourse(null)}>
            <Text style={styles.changeCourse}>更换课程</Text>
          </TouchableOpacity>
        </View>

        <Text style={styles.label}>标题</Text>
        <TextInput
          value={title}
          onChangeText={setTitle}
          placeholder="一句话说清楚这门课的体验"
          placeholderTextColor={colors.muted}
          style={styles.input}
          maxLength={120}
        />

        <Text style={styles.label}>正文</Text>
        <TextInput
          value={body}
          onChangeText={(value) => setBody(value.slice(0, MAX_BODY))}
          placeholder="上课方式、作业量、给分、考核、避雷点…"
          placeholderTextColor={colors.muted}
          style={[styles.input, styles.textarea]}
          multiline
          textAlignVertical="top"
        />
        <Text style={styles.counter}>
          {body.length}/{MAX_BODY}
        </Text>

        <Text style={styles.label}>修读年份（可选）</Text>
        <TouchableOpacity style={[styles.input, { justifyContent: 'center' }]}
          accessibilityRole="button" accessibilityLabel={`修读年份，${year === 'unspecified' ? '未注明' : year}`}
          onPress={() => setYearPicker(true)}>
          <Text style={{ color: colors.ink }}>{year === 'unspecified' ? '年份未注明' : year} ▾</Text>
        </TouchableOpacity>
        <AttendanceYearPicker visible={yearPicker} value={year} onClose={() => setYearPicker(false)}
          onChange={(value) => { setYear(value); setYearPicker(false); }} />

        <Text style={styles.label}>话题标签（最多 10 个）</Text>
        {topics.isError && <TouchableOpacity onPress={() => void topics.refetch()}>
          <Text style={styles.changeCourse}>话题暂时无法加载，点击重试</Text>
        </TouchableOpacity>}
        <View style={styles.tagWrap}>
          {(topics.data?.items ?? []).map((item) => {
            const active = selectedTags.includes(item.name);
            return (
              <TouchableOpacity
                key={item.id}
                accessibilityRole="button"
                accessibilityState={{ selected: active }}
                style={[styles.tagChip, active && styles.tagChipActive]}
                onPress={() => {
                  if (!active && selectedTags.length >= 10) return;
                  toggleTag(item.name);
                }}
              >
                <Text style={[styles.tagText, active && styles.tagTextActive]}>{item.name}</Text>
              </TouchableOpacity>
            );
          })}
        </View>

        <Text style={styles.label}>图片（最多 {MAX_IMAGES} 张）</Text>
        {operation === 'images' && <Text accessibilityLiveRegion="polite" style={styles.privacy}>正在上传图片（已添加 {images.length} 张）</Text>}
        <Text style={styles.privacy}>支持 JPEG 和 PNG，每张最多 8 MB。请先用相册编辑工具遮挡姓名、学号和聊天对象等个人信息。</Text>
        <View style={styles.imageRow}>
          {images.map((uri) => (
            <View key={uri} style={styles.imageBox}>
              <Image source={{ uri: `${apiBaseUrl}${uri}`, headers: { Authorization: `Bearer ${token}` } }} style={styles.image} />
              <TouchableOpacity style={styles.removeImage} accessibilityRole="button"
                accessibilityLabel={`移除第 ${images.indexOf(uri) + 1} 张图片`} disabled={busy} onPress={() => setImages((value) => value.filter((item) => item !== uri))}>
                <Text style={styles.removeImageText}>✕</Text>
              </TouchableOpacity>
            </View>
          ))}
          {images.length < MAX_IMAGES ? (
            <TouchableOpacity style={[styles.imageBox, styles.addImage]} disabled={busy} accessibilityRole="button" accessibilityLabel="添加图片" onPress={() => void pickImages()}>
              <Text style={styles.addImageText}>＋</Text>
            </TouchableOpacity>
          ) : null}
        </View>

        <TouchableOpacity style={styles.anonymousRow} accessibilityRole="checkbox" accessibilityState={{ checked: anonymous }} onPress={() => setAnonymous((value) => !value)}>
          <View style={[styles.checkbox, anonymous && styles.checkboxActive]}>
            {anonymous ? <Text style={styles.checkmark}>✓</Text> : null}
          </View>
          <View style={{ flex: 1 }}>
            <Text style={styles.anonymousTitle}>匿名发布</Text>
            <Text style={styles.anonymousHint}>其他同学只会看到「同校已验证」；平台会保留账号关联，用于举报处理等必要用途。</Text>
          </View>
        </TouchableOpacity>

        {!!policy.data?.content_review_sla_hours &&
          <Text style={styles.privacy}>预计审核时限 {policy.data.content_review_sla_hours} 小时。</Text>}
        {!!policy.data?.community_guidelines && <Text style={styles.privacy}>{policy.data.community_guidelines}</Text>}

        <TouchableOpacity style={[styles.submit, busy && styles.submitDisabled]} onPress={() => void submit()} disabled={busy}>
          {busy ? <ActivityIndicator color="#fff" /> : <Text style={styles.submitText}>提交审核</Text>}
        </TouchableOpacity>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: colors.paper },
  header: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', paddingHorizontal: 16, paddingVertical: 12 },
  back: { color: colors.ink, fontSize: 26, paddingHorizontal: 4 },
  headerTitle: { color: colors.ink, fontSize: 16, fontWeight: '800' },
  courseCode: { color: colors.green, fontSize: 12, fontWeight: '800' },
  courseName: { color: colors.ink, fontSize: 14, fontWeight: '700' },
  courseCard: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#fff',
    borderRadius: 14,
    borderWidth: 1,
    borderColor: colors.line,
    padding: 13,
    marginHorizontal: 16,
    marginTop: 6,
  },
  changeCourse: { color: colors.green, fontSize: 11, fontWeight: '700' },
  label: { color: colors.ink, fontSize: 12, fontWeight: '800', marginHorizontal: 16, marginTop: 16, marginBottom: 6 },
  input: {
    marginHorizontal: 16,
    backgroundColor: '#fff',
    borderWidth: 1,
    borderColor: colors.line,
    borderRadius: 12,
    paddingHorizontal: 13,
    paddingVertical: 11,
    color: colors.ink,
    fontSize: 13,
  },
  textarea: { minHeight: 150, lineHeight: 19 },
  counter: { color: colors.faint, fontSize: 10, textAlign: 'right', marginHorizontal: 18, marginTop: 4 },
  tagWrap: { flexDirection: 'row', flexWrap: 'wrap', gap: 7, marginHorizontal: 16 },
  tagChip: { borderRadius: 999, borderWidth: 1, borderColor: colors.line, backgroundColor: '#fff', paddingHorizontal: 11, paddingVertical: 6 },
  tagChipActive: { backgroundColor: colors.ink, borderColor: colors.ink },
  tagText: { color: colors.muted, fontSize: 11 },
  tagTextActive: { color: '#fff', fontWeight: '700' },
  privacy: { color: '#8a7a4a', fontSize: 10, lineHeight: 15, marginHorizontal: 16, marginBottom: 8 },
  imageRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 8, marginHorizontal: 16 },
  imageBox: { width: 78, height: 78, borderRadius: 11, overflow: 'hidden', backgroundColor: '#eceee7' },
  image: { width: '100%', height: '100%' },
  removeImage: { position: 'absolute', top: 0, right: 0, width: 48, height: 48, alignItems: 'flex-end', padding: 3 },
  removeImageText: { width: 20, height: 20, borderRadius: 10, textAlign: 'center', lineHeight: 20, overflow: 'hidden', color: '#fff', backgroundColor: '#00000099', fontSize: 12, fontWeight: '800' },
  addImage: { alignItems: 'center', justifyContent: 'center', borderWidth: 1, borderColor: colors.line, borderStyle: 'dashed' },
  addImageText: { color: colors.muted, fontSize: 24 },
  anonymousRow: { flexDirection: 'row', alignItems: 'center', gap: 10, marginHorizontal: 16, marginTop: 18 },
  checkbox: { width: 20, height: 20, borderRadius: 6, borderWidth: 1, borderColor: colors.line, backgroundColor: '#fff', alignItems: 'center', justifyContent: 'center' },
  checkboxActive: { backgroundColor: colors.green, borderColor: colors.green },
  checkmark: { color: '#fff', fontSize: 12, fontWeight: '800' },
  anonymousTitle: { color: colors.ink, fontSize: 12, fontWeight: '700' },
  anonymousHint: { color: colors.muted, fontSize: 10, lineHeight: 15, marginTop: 2 },
  submit: { marginHorizontal: 16, marginTop: 22, backgroundColor: colors.green, borderRadius: 13, paddingVertical: 14, alignItems: 'center' },
  submitDisabled: { opacity: 0.6 },
  submitText: { color: '#fff', fontSize: 13, fontWeight: '800' },
});
