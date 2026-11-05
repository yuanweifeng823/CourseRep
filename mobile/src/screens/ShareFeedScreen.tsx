import { useEffect, useMemo, useState } from 'react';
import { ActivityIndicator, FlatList, Image, ScrollView, StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { apiBaseUrl, apiFetch } from '../../api';
import type { CourseFacets, FeedOutput, FeedPost, TopicTagListResponse, UserProfile } from '../api/types';
import { DepartmentPicker } from '../ui/DepartmentPicker';
import { colors } from '../ui/theme';

type Props = {
  token: string;
  onOpenPost: (post: FeedPost) => void;
  onCompose: () => void;
};

const PAGE_SIZE = 10;
const ALL_DEPARTMENTS = '全部专业';

export function ShareFeedScreen({ token, onOpenPost, onCompose }: Props) {
  const [department, setDepartment] = useState<string | null>(null);
  const [tag, setTag] = useState('');
  const [sort, setSort] = useState<'newest' | 'popular'>('newest');
  const [pickerOpen, setPickerOpen] = useState(false);

  const profile = useQuery({
    queryKey: ['profile'],
    queryFn: async () => {
      const response = await apiFetch('/api/v1/users/me', token);
      if (!response.ok) throw new Error('读取个人资料失败');
      return (await response.json()) as UserProfile;
    },
  });
  const facets = useQuery({
    queryKey: ['course-facets'],
    queryFn: async () => {
      const response = await apiFetch('/api/v1/course-facets', token);
      if (!response.ok) throw new Error('读取专业列表失败');
      return (await response.json()) as CourseFacets;
    },
  });
  const tags = useQuery({
    queryKey: ['topic-tags'],
    queryFn: async () => {
      const response = await apiFetch('/api/v1/topic-tags', token);
      if (!response.ok) throw new Error('读取话题失败');
      return (await response.json()) as TopicTagListResponse;
    },
  });

  useEffect(() => {
    if (department === null && profile.data) setDepartment(profile.data.department ?? '');
  }, [profile.data, department]);

  const feed = useInfiniteQuery({
    queryKey: ['feed', department, tag, sort],
    enabled: department !== null,
    initialPageParam: 1,
    queryFn: async ({ pageParam }) => {
      const params = new URLSearchParams({ page: String(pageParam), page_size: String(PAGE_SIZE), sort });
      if (department) params.set('department', department);
      if (tag) params.set('tag', tag);
      const response = await apiFetch(`/api/v1/posts?${params.toString()}`, token);
      if (!response.ok) throw new Error('读取经验帖失败');
      return (await response.json()) as FeedOutput;
    },
    getNextPageParam: (lastPage, allPages) => (lastPage.has_more ? allPages.length + 1 : undefined),
  });

  const items = useMemo(() => feed.data?.pages.flatMap((page) => page.items ?? []) ?? [], [feed.data]);
  const departmentOptions = useMemo(() => [ALL_DEPARTMENTS, ...(facets.data?.departments ?? [])], [facets.data]);

  const header = (
    <View>
      <View style={styles.headingRow}>
        <View>
          <Text style={styles.heading}>分享</Text>
          <Text style={styles.subheading}>{department ? `${department} 的修课经验` : '全部专业的修课经验'}</Text>
        </View>
        <View style={styles.sortGroup}>
          {(['newest', 'popular'] as const).map((value) => (
            <TouchableOpacity key={value} onPress={() => setSort(value)} style={[styles.sortButton, sort === value && styles.sortButtonActive]}>
              <Text style={[styles.sortText, sort === value && styles.sortTextActive]}>{value === 'newest' ? '最新' : '最热'}</Text>
            </TouchableOpacity>
          ))}
        </View>
      </View>

      <View style={styles.filterRow}>
        <TouchableOpacity style={styles.departmentChip} onPress={() => setPickerOpen(true)}>
          <Text style={styles.departmentChipText}>{department ? department : ALL_DEPARTMENTS} ▾</Text>
        </TouchableOpacity>
        <Text style={styles.countText}>{items.length ? `${items.length} 篇` : ''}</Text>
      </View>

      <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.tagRow}>
        {['', ...((tags.data?.items ?? []).map((item) => item.name))].map((name) => {
          const active = tag === name;
          return (
            <TouchableOpacity key={name || 'all'} onPress={() => setTag(name)} style={[styles.tagChip, active && styles.tagChipActive]}>
              <Text style={[styles.tagText, active && styles.tagTextActive]}>{name || '全部话题'}</Text>
            </TouchableOpacity>
          );
        })}
      </ScrollView>
    </View>
  );

  const empty = !feed.isLoading && items.length === 0;

  return (
    <View style={styles.container}>
      <FlatList
        data={items}
        keyExtractor={(item) => String(item.id)}
        numColumns={2}
        columnWrapperStyle={styles.column}
        contentContainerStyle={styles.listContent}
        ListHeaderComponent={header}
        renderItem={({ item }) => <FeedCard item={item} token={token} onPress={() => onOpenPost(item)} />}
        onEndReachedThreshold={0.4}
        onEndReached={() => {
          if (feed.hasNextPage && !feed.isFetchingNextPage) void feed.fetchNextPage();
        }}
        ListEmptyComponent={
          feed.isLoading ? (
            <ActivityIndicator color={colors.green} style={{ marginTop: 36 }} />
          ) : feed.isError ? (
            <Text style={styles.empty}>经验帖暂时无法加载，稍后再试。</Text>
          ) : empty ? (
            <View style={styles.emptyBox}>
              <Text style={styles.emptyTitle}>{department ? `${department} 还没有经验帖` : '还没有经验帖'}</Text>
              <Text style={styles.emptyBody}>写下第一篇修课经验，审核通过后会出现在这里。</Text>
              <TouchableOpacity style={styles.primary} onPress={onCompose}>
                <Text style={styles.primaryText}>写经验帖</Text>
              </TouchableOpacity>
            </View>
          ) : null
        }
        ListFooterComponent={
          feed.isFetchingNextPage ? <ActivityIndicator color={colors.green} style={{ marginVertical: 18 }} /> : null
        }
      />
      <TouchableOpacity style={styles.fab} onPress={onCompose} accessibilityLabel="写经验帖">
        <Text style={styles.fabIcon}>＋</Text>
        <Text style={styles.fabText}>发帖</Text>
      </TouchableOpacity>
      <DepartmentPicker
        visible={pickerOpen}
        departments={departmentOptions}
        current={department ? department : ALL_DEPARTMENTS}
        title="筛选专业"
        hint="默认展示你所在专业的经验帖，也可以查看全部专业。"
        onSelect={(value) => {
          setDepartment(value === ALL_DEPARTMENTS ? '' : value);
          setPickerOpen(false);
        }}
        onClose={() => setPickerOpen(false)}
      />
    </View>
  );
}

function FeedCard({ item, token, onPress }: { item: FeedPost; token: string; onPress: () => void }) {
  const cover = item.image_urls?.[0];
  return (
    <TouchableOpacity style={styles.card} activeOpacity={0.86} onPress={onPress}>
      {cover ? (
        <Image
          source={{ uri: `${apiBaseUrl}${cover}`, headers: { Authorization: `Bearer ${token}` } }}
          style={styles.cover}
          resizeMode="cover"
        />
      ) : (
        <View style={[styles.cover, styles.coverFallback]}>
          <Text style={styles.coverTag} numberOfLines={1}>
            {item.topic_tags?.[0] ?? '修课经验'}
          </Text>
          <Text style={styles.coverCode}>{item.course.course_code}</Text>
        </View>
      )}
      <Text style={styles.cardTitle} numberOfLines={2}>
        {item.title}
      </Text>
      {!cover && (
        <Text style={styles.cardExcerpt} numberOfLines={2}>
          {item.body}
        </Text>
      )}
      <View style={styles.cardMeta}>
        <Text style={styles.metaText} numberOfLines={1}>
          {item.course.course_code}
        </Text>
        <Text style={styles.metaText}>♥ {item.like_count}</Text>
      </View>
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: colors.paper },
  listContent: { paddingHorizontal: 16, paddingBottom: 110 },
  column: { gap: 12 },
  headingRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', paddingTop: 6, paddingBottom: 2 },
  heading: { color: colors.ink, fontSize: 25, fontWeight: '800', letterSpacing: -0.6 },
  subheading: { color: colors.muted, fontSize: 11, marginTop: 3 },
  sortGroup: { flexDirection: 'row', backgroundColor: '#fff', borderRadius: 12, borderWidth: 1, borderColor: colors.line, padding: 3 },
  sortButton: { paddingHorizontal: 12, paddingVertical: 6, borderRadius: 9 },
  sortButtonActive: { backgroundColor: colors.green },
  sortText: { color: colors.muted, fontSize: 11, fontWeight: '700' },
  sortTextActive: { color: '#fff' },
  filterRow: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', marginTop: 12 },
  departmentChip: {
    backgroundColor: '#fff',
    borderRadius: 11,
    borderWidth: 1,
    borderColor: colors.line,
    paddingHorizontal: 13,
    paddingVertical: 8,
  },
  departmentChipText: { color: colors.green, fontSize: 12, fontWeight: '800' },
  countText: { color: '#9aa39c', fontSize: 10 },
  tagRow: { gap: 7, paddingVertical: 10 },
  tagChip: {
    borderRadius: 999,
    borderWidth: 1,
    borderColor: colors.line,
    backgroundColor: '#fff',
    paddingHorizontal: 12,
    paddingVertical: 6,
  },
  tagChipActive: { backgroundColor: colors.ink, borderColor: colors.ink },
  tagText: { color: colors.muted, fontSize: 11 },
  tagTextActive: { color: '#fff', fontWeight: '700' },
  card: { flex: 1, marginBottom: 14, backgroundColor: '#fff', borderRadius: 16, borderWidth: 1, borderColor: colors.line, overflow: 'hidden' },
  cover: { width: '100%', aspectRatio: 0.86, backgroundColor: '#e9ece4' },
  coverFallback: { justifyContent: 'space-between', padding: 12, backgroundColor: '#dfe7dc' },
  coverTag: { color: colors.green, fontSize: 10, fontWeight: '800' },
  coverCode: { color: colors.ink, fontSize: 20, fontWeight: '800', letterSpacing: -0.5 },
  cardTitle: { color: colors.ink, fontSize: 13, fontWeight: '700', lineHeight: 18, paddingHorizontal: 10, paddingTop: 9 },
  cardExcerpt: { color: colors.muted, fontSize: 10, lineHeight: 15, paddingHorizontal: 10, paddingTop: 3 },
  cardMeta: { flexDirection: 'row', justifyContent: 'space-between', paddingHorizontal: 10, paddingVertical: 9 },
  metaText: { color: '#8d968f', fontSize: 10 },
  emptyBox: { alignItems: 'center', paddingTop: 40, gap: 8 },
  emptyTitle: { color: colors.ink, fontSize: 16, fontWeight: '800' },
  emptyBody: { color: colors.muted, fontSize: 12, textAlign: 'center', lineHeight: 18 },
  empty: { color: colors.muted, textAlign: 'center', marginTop: 30, fontSize: 12 },
  primary: { backgroundColor: colors.green, borderRadius: 12, paddingHorizontal: 20, paddingVertical: 11, marginTop: 8 },
  primaryText: { color: '#fff', fontSize: 12, fontWeight: '800' },
  fab: {
    position: 'absolute',
    right: 16,
    bottom: 18,
    backgroundColor: colors.green,
    borderRadius: 999,
    paddingHorizontal: 18,
    paddingVertical: 13,
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    shadowColor: '#000',
    shadowOpacity: 0.18,
    shadowRadius: 8,
    shadowOffset: { width: 0, height: 4 },
    elevation: 5,
  },
  fabIcon: { color: colors.lime, fontSize: 16, fontWeight: '800' },
  fabText: { color: '#fff', fontSize: 13, fontWeight: '800' },
});
