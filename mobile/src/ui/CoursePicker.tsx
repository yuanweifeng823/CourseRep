import { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ActivityIndicator, FlatList, StyleSheet, Text, TextInput, TouchableOpacity, View } from 'react-native';
import { apiFetch } from '../../api';
import type { Course, CourseListResponse } from '../api/types';
import { colors } from './theme';

type Props = {
  token: string;
  onClose: () => void;
  onSelect: (course: Course) => void;
};

export function CoursePicker({ token, onClose, onSelect }: Props) {
  const [search, setSearch] = useState('');
  const [query, setQuery] = useState('');
  useEffect(() => {
    const timer = setTimeout(() => setQuery(search.trim()), 300);
    return () => clearTimeout(timer);
  }, [search]);

  const courses = useQuery({
    queryKey: ['course-picker', query],
    queryFn: async () => {
      const params = new URLSearchParams({ page_size: '20' });
      if (query) params.set('q', query);
      const response = await apiFetch(`/api/v1/courses?${params}`, token);
      if (!response.ok) throw new Error('读取课程失败');
      return (await response.json()) as CourseListResponse;
    },
  });

  return (
    <View style={styles.container}>
      <View style={styles.header}>
        <TouchableOpacity onPress={onClose} hitSlop={12} accessibilityRole="button" accessibilityLabel="返回">
          <Text style={styles.back}>‹</Text>
        </TouchableOpacity>
        <Text style={styles.headerTitle}>选择课程</Text>
        <View style={{ width: 24 }} />
      </View>
      <TextInput value={search} onChangeText={setSearch} placeholder="搜索课程代码或名称"
        accessibilityLabel="搜索课程" style={styles.search} autoCorrect={false} returnKeyType="search" />
      <FlatList
        data={courses.data?.items ?? []}
        keyExtractor={(item) => String(item.id)}
        keyboardShouldPersistTaps="handled"
        renderItem={({ item }) => (
          <TouchableOpacity style={styles.row} accessibilityRole="button" onPress={() => onSelect(item)}>
            <Text style={styles.code}>{item.course_code}</Text>
            <Text style={styles.name}>{item.course_name}</Text>
            <Text style={styles.meta}>{item.professor || '教授待补充'} · {item.semester}</Text>
          </TouchableOpacity>
        )}
        ListEmptyComponent={courses.isPending ? <ActivityIndicator color={colors.green} /> :
          courses.isError ? <TouchableOpacity onPress={() => void courses.refetch()}>
            <Text style={styles.empty}>课程暂时无法加载，点击重试</Text>
          </TouchableOpacity> : <Text style={styles.empty}>{query ? '没有匹配的课程' : '课程目录暂无课程'}</Text>}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: colors.paper },
  header: { height: 54, flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', paddingHorizontal: 20 },
  headerTitle: { color: colors.ink, fontSize: 17, fontWeight: '800' },
  back: { color: colors.ink, fontSize: 26, paddingHorizontal: 4 },
  search: { margin: 16, padding: 12, borderRadius: 12, color: colors.ink, backgroundColor: colors.card, borderColor: colors.line, borderWidth: 1 },
  row: { paddingVertical: 12, paddingHorizontal: 20, borderBottomWidth: 1, borderColor: colors.line },
  code: { color: colors.green, fontSize: 13, fontWeight: '800' },
  name: { color: colors.ink, fontSize: 15, fontWeight: '700' },
  meta: { color: colors.muted, fontSize: 12, marginTop: 4 },
  empty: { color: colors.muted, textAlign: 'center', marginVertical: 20 },
});
