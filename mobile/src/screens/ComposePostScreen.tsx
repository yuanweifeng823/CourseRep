import { useEffect, useMemo, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  FlatList,
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
import type { Course, CourseListResponse, TopicTagListResponse } from '../api/types';
import { colors } from '../ui/theme';

type Props = {
  token: string;
  onClose: () => void;
  onPublished: () => void;
};

const MAX_IMAGES = 9;
const MAX_BODY = 2000;

export function ComposePostScreen({ token, onClose, onPublished }: Props) {
  const queryClient = useQueryClient();
  const [course, setCourse] = useState<Course | null>(null);
  const [search, setSearch] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  const [year, setYear] = useState('');
  const [anonymous, setAnonymous] = useState(false);
  const [images, setImages] = useState<string[]>([]);
  const [selectedTags, setSelectedTags] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(search.trim()), 300);
    return () => clearTimeout(timer);
  }, [search]);

  const courses = useQuery({
    queryKey: ['compose-courses', debouncedSearch],
    enabled: course === null,
    queryFn: async () => {
      const params = new URLSearchParams({ page: '1', page_size: '20' });
      if (debouncedSearch) params.set('q', debouncedSearch);
      const response = await apiFetch(`/api/v1/courses?${params.toString()}`, token);
      if (!response.ok) throw new Error('读取课程失败');
      return (await response.json()) as CourseListResponse;
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

  const courseItems = useMemo(() => courses.data?.items ?? [], [courses.data]);

  const pickImages = async () => {
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ['images'],
      allowsMultipleSelection: true,
      selectionLimit: MAX_IMAGES - images.length,
      quality: 0.9,
    });
    if (result.canceled) return;
    setBusy(true);
    try {
      for (const asset of result.assets) {
        const form = new FormData();
        form.append('image', {
          uri: asset.uri,
          name: asset.fileName || 'photo.jpg',
          type: asset.mimeType || 'image/jpeg',
        } as unknown as Blob);
        const response = await apiFetch('/api/v1/uploads', token, { method: 'POST', body: form });
        const data = (await response.json()) as { image_url?: string; error?: string };
        if (!response.ok || !data.image_url) throw new Error(data.error || '图片上传失败');
        setImages((value) => [...value, data.image_url!]);
      }
    } catch (error) {
      Alert.alert('图片上传失败', error instanceof Error ? error.message : '请稍后再试');
    } finally {
      setBusy(false);
    }
  };

  const toggleTag = (name: string) => {
    setSelectedTags((value) => (value.includes(name) ? value.filter((item) => item !== name) : [...value, name]));
  };

  const submit = async () => {
    if (!course) return;
    const currentYear = new Date().getFullYear();
    const yearValue = year.trim() ? Number(year.trim()) : null;
    if (yearValue !== null && (!Number.isInteger(yearValue) || yearValue < 1990 || yearValue > currentYear)) {
      Alert.alert('年份无效', `请填写 1990 到 ${currentYear} 之间的年份，或留空表示年份未注明。`);
      return;
    }
    if (!title.trim()) return Alert.alert('需要标题', '请填写经验帖标题。');
    if (!body.trim()) return Alert.alert('需要正文', '请填写经验帖正文。');
    setBusy(true);
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
      queryClient.invalidateQueries({ queryKey: ['feed'] });
      Alert.alert('已提交', '审核通过后会出现在分享页。', [{ text: '好', onPress: onPublished }]);
    } catch (error) {
      Alert.alert('无法发布', error instanceof Error ? error.message : '请稍后再试');
    } finally {
      setBusy(false);
    }
  };

  if (!course) {
    return (
      <View style={styles.container}>
        <View style={styles.header}>
          <TouchableOpacity onPress={onClose} accessibilityLabel="取消">
            <Text style={styles.back}>‹</Text>
          </TouchableOpacity>
          <Text style={styles.headerTitle}>选择课程</Text>
          <View style={{ width: 24 }} />
        </View>
        <Text style={styles.hint}>经验帖需要关联一门课程，先搜索课程代码或名称。</Text>
        <TextInput
          value={search}
          onChangeText={setSearch}
          placeholder="例如 EE5607、Machine Learning"
          placeholderTextColor="#9aa39c"
          style={styles.search}
          autoCorrect={false}
        />
        {courses.isLoading ? (
          <ActivityIndicator color={colors.green} style={{ marginTop: 28 }} />
        ) : (
          <FlatList
            data={courseItems}
            keyExtractor={(item) => String(item.id)}
            contentContainerStyle={{ paddingBottom: 30 }}
            keyboardShouldPersistTaps="handled"
            renderItem={({ item }) => (
              <TouchableOpacity style={styles.courseRow} onPress={() => setCourse(item)}>
                <View style={{ flex: 1 }}>
                  <Text style={styles.courseCode}>{item.course_code}</Text>
                  <Text style={styles.courseName} numberOfLines={2}>
                    {item.course_name}
                  </Text>
                  <Text style={styles.courseMeta}>
                    {item.department} · {item.semester}
                  </Text>
                </View>
                <Text style={styles.chevron}>›</Text>
              </TouchableOpacity>
            )}
            ListEmptyComponent={
              courses.isError ? (
                <Text style={styles.empty}>课程暂时无法加载，稍后再试。</Text>
              ) : (
                <Text style={styles.empty}>没有找到相关课程</Text>
              )
            }
          />
        )}
      </View>
    );
  }

  return (
    <KeyboardAvoidingView style={styles.container} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
      <View style={styles.header}>
        <TouchableOpacity onPress={onClose} accessibilityLabel="取消">
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
          <TouchableOpacity onPress={() => setCourse(null)}>
            <Text style={styles.changeCourse}>更换课程</Text>
          </TouchableOpacity>
        </View>

        <Text style={styles.label}>标题</Text>
        <TextInput
          value={title}
          onChangeText={setTitle}
          placeholder="一句话说清楚这门课的体验"
          placeholderTextColor="#9aa39c"
          style={styles.input}
          maxLength={120}
        />

        <Text style={styles.label}>正文</Text>
        <TextInput
          value={body}
          onChangeText={(value) => setBody(value.slice(0, MAX_BODY))}
          placeholder="上课方式、作业量、给分、考核、避雷点…"
          placeholderTextColor="#9aa39c"
          style={[styles.input, styles.textarea]}
          multiline
          textAlignVertical="top"
        />
        <Text style={styles.counter}>
          {body.length}/{MAX_BODY}
        </Text>

        <Text style={styles.label}>修读年份（可选）</Text>
        <TextInput
          value={year}
          onChangeText={setYear}
          placeholder={`如 ${new Date().getFullYear() - 1}，留空表示年份未注明`}
          placeholderTextColor="#9aa39c"
          style={styles.input}
          keyboardType="number-pad"
          maxLength={4}
        />

        <Text style={styles.label}>话题标签（最多 10 个）</Text>
        <View style={styles.tagWrap}>
          {(topics.data?.items ?? []).map((item) => {
            const active = selectedTags.includes(item.name);
            return (
              <TouchableOpacity
                key={item.id}
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
        <Text style={styles.privacy}>发布前请遮挡图片中的姓名、学号和聊天对象等个人信息。</Text>
        <View style={styles.imageRow}>
          {images.map((uri) => (
            <View key={uri} style={styles.imageBox}>
              <Image source={{ uri: `${apiBaseUrl}${uri}`, headers: { Authorization: `Bearer ${token}` } }} style={styles.image} />
              <TouchableOpacity style={styles.removeImage} onPress={() => setImages((value) => value.filter((item) => item !== uri))}>
                <Text style={styles.removeImageText}>✕</Text>
              </TouchableOpacity>
            </View>
          ))}
          {images.length < MAX_IMAGES ? (
            <TouchableOpacity style={[styles.imageBox, styles.addImage]} onPress={() => void pickImages()}>
              <Text style={styles.addImageText}>＋</Text>
            </TouchableOpacity>
          ) : null}
        </View>

        <TouchableOpacity style={styles.anonymousRow} onPress={() => setAnonymous((value) => !value)}>
          <View style={[styles.checkbox, anonymous && styles.checkboxActive]}>
            {anonymous ? <Text style={styles.checkmark}>✓</Text> : null}
          </View>
          <View style={{ flex: 1 }}>
            <Text style={styles.anonymousTitle}>匿名发布</Text>
            <Text style={styles.anonymousHint}>其他同学只会看到「同校已验证」；评奖与举报处理仍需账号关联。</Text>
          </View>
        </TouchableOpacity>

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
  hint: { color: colors.muted, fontSize: 12, lineHeight: 18, paddingHorizontal: 16 },
  search: {
    marginHorizontal: 16,
    marginTop: 12,
    backgroundColor: '#fff',
    borderWidth: 1,
    borderColor: colors.line,
    borderRadius: 12,
    paddingHorizontal: 13,
    paddingVertical: 10,
    color: colors.ink,
    fontSize: 13,
  },
  courseRow: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#fff',
    borderWidth: 1,
    borderColor: colors.line,
    borderRadius: 14,
    padding: 13,
    marginHorizontal: 16,
    marginTop: 9,
  },
  courseCode: { color: colors.green, fontSize: 12, fontWeight: '800' },
  courseName: { color: colors.ink, fontSize: 13, fontWeight: '700', marginTop: 2 },
  courseMeta: { color: colors.muted, fontSize: 10, marginTop: 3 },
  chevron: { color: colors.faint, fontSize: 20 },
  empty: { color: colors.muted, textAlign: 'center', marginTop: 24, fontSize: 12 },
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
  removeImage: { position: 'absolute', top: 3, right: 3, backgroundColor: '#00000088', borderRadius: 999, width: 18, height: 18, alignItems: 'center', justifyContent: 'center' },
  removeImageText: { color: '#fff', fontSize: 10 },
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
