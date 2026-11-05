import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { ActivityIndicator, ScrollView, StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { apiFetch } from '../../api';
import type { CourseFacets, UserProfile } from '../../src/api/types';
import type { MyListKind } from './MyListScreen';
import { DepartmentPicker } from '../ui/DepartmentPicker';
import { ServerField } from '../ui/ServerField';
import { colors, statusLabel } from '../ui/theme';

type Props = {
  token: string;
  onLogout: () => void;
  onOpenList: (kind: MyListKind) => void;
  onOpenAdmin: () => void;
};

const entries: { kind: MyListKind; icon: string; title: string; hint: string }[] = [
  { kind: 'courses', icon: '✓', title: '已修读课程', hint: '标记过修读的课程' },
  { kind: 'reviews', icon: '★', title: '我的评价', hint: '课程评分与文字评价' },
  { kind: 'posts', icon: '✎', title: '我的经验帖', hint: '修课经验与审核状态' },
  { kind: 'bookmarks', icon: '🔖', title: '我的收藏', hint: '收藏的经验帖' },
];

export function MeScreen({ token, onLogout, onOpenList, onOpenAdmin }: Props) {
  const queryClient = useQueryClient();
  const [pickerOpen, setPickerOpen] = useState(false);
  const [message, setMessage] = useState('');

  const profile = useQuery({
    queryKey: ['profile'],
    queryFn: async () => {
      const response = await apiFetch('/api/v1/users/me', token);
      if (!response.ok) throw new Error('个人资料暂时无法加载');
      return (await response.json()) as UserProfile;
    },
  });

  const facets = useQuery({
    queryKey: ['course-facets'],
    queryFn: async () => {
      const response = await apiFetch('/api/v1/course-facets', token);
      if (!response.ok) throw new Error();
      return (await response.json()) as CourseFacets;
    },
  });

  const changeDepartment = async (department: string) => {
    setMessage('');
    const response = await apiFetch('/api/v1/users/me', token, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ department }),
    });
    if (!response.ok) {
      setMessage('保存失败，请稍后再试');
      return;
    }
    setPickerOpen(false);
    setMessage('专业已更新');
    await queryClient.invalidateQueries({ queryKey: ['profile'] });
    await queryClient.invalidateQueries({ queryKey: ['home'] });
  };

  const isAdmin = profile.data?.role === 'admin';

  return (
    <View style={{ flex: 1 }}>
      <ScrollView contentContainerStyle={styles.content}>
        <Text style={styles.eyebrow}>MY ACCOUNT</Text>
        <Text style={styles.title}>我的</Text>

        <View style={styles.profileCard}>
          {profile.isLoading ? (
            <ActivityIndicator color={colors.green} />
          ) : (
            <>
              <View style={styles.avatar}>
                <Text style={styles.avatarText}>{(profile.data?.nickname ?? 'C').slice(0, 1)}</Text>
              </View>
              <Text style={styles.nickname}>{profile.data?.nickname ?? '未命名同学'}</Text>
              <Text style={styles.meta}>
                {statusLabel(profile.data?.status)} · {profile.data?.role === 'admin' ? '运营管理员' : '普通同学'}
              </Text>
              <TouchableOpacity style={styles.deptRow} onPress={() => setPickerOpen(true)}>
                <Text style={styles.deptText}>专业：{profile.data?.department || '尚未选择'}</Text>
                <Text style={styles.deptAction}>修改</Text>
              </TouchableOpacity>
              {!!message && <Text style={styles.message}>{message}</Text>}
            </>
          )}
        </View>

        <View style={styles.group}>
          {entries.map((entry) => (
            <TouchableOpacity key={entry.kind} style={styles.row} onPress={() => onOpenList(entry.kind)}>
              <Text style={styles.rowIcon}>{entry.icon}</Text>
              <View style={{ flex: 1 }}>
                <Text style={styles.rowTitle}>{entry.title}</Text>
                <Text style={styles.rowHint}>{entry.hint}</Text>
              </View>
              <Text style={styles.rowArrow}>›</Text>
            </TouchableOpacity>
          ))}
        </View>

        {isAdmin ? (
          <TouchableOpacity style={styles.adminButton} onPress={onOpenAdmin}>
            <Text style={styles.adminIcon}>⚙</Text>
            <View style={{ flex: 1 }}>
              <Text style={styles.adminTitle}>审核管理</Text>
              <Text style={styles.adminHint}>评价、经验帖、举报、纠错与课程导入</Text>
            </View>
            <Text style={styles.adminArrow}>›</Text>
          </TouchableOpacity>
        ) : (
          <View style={styles.lockedCard}>
            <Text style={styles.lockedTitle}>运营管理</Text>
            <Text style={styles.lockedHint}>仅管理员账号可见。普通账号不能进入审核队列。</Text>
          </View>
        )}

        <ServerField />

        <TouchableOpacity style={styles.logout} onPress={onLogout}>
          <Text style={styles.logoutText}>退出登录</Text>
        </TouchableOpacity>
        <Text style={styles.footer}>CourseRep · CityU 课程经验社区</Text>
      </ScrollView>

      <DepartmentPicker
        visible={pickerOpen}
        departments={facets.data?.departments ?? []}
        current={profile.data?.department}
        title="修改专业"
        hint="首页推荐与课程页的专业筛选会随之更新。"
        onSelect={(department) => void changeDepartment(department)}
        onClose={() => setPickerOpen(false)}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  content: { paddingHorizontal: 20, paddingBottom: 40 },
  eyebrow: { color: colors.green, fontSize: 10, fontWeight: '800', letterSpacing: 1.6, marginTop: 6 },
  title: { color: colors.ink, fontSize: 27, fontWeight: '800', marginTop: 6, letterSpacing: -0.6 },
  profileCard: {
    marginTop: 16,
    backgroundColor: colors.green,
    borderRadius: 20,
    padding: 20,
    alignItems: 'flex-start',
  },
  avatar: {
    width: 46,
    height: 46,
    borderRadius: 16,
    backgroundColor: '#ffffff22',
    alignItems: 'center',
    justifyContent: 'center',
  },
  avatarText: { color: colors.lime, fontSize: 22, fontWeight: '800' },
  nickname: { color: '#fff', fontSize: 19, fontWeight: '800', marginTop: 12 },
  meta: { color: '#d2ded5', fontSize: 11, marginTop: 5 },
  deptRow: {
    marginTop: 14,
    alignSelf: 'stretch',
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    backgroundColor: '#ffffff1c',
    borderRadius: 12,
    paddingHorizontal: 12,
    paddingVertical: 10,
  },
  deptText: { color: '#fff', fontSize: 12, fontWeight: '700' },
  deptAction: { color: colors.lime, fontSize: 11, fontWeight: '800' },
  message: { color: colors.lime, fontSize: 11, marginTop: 8 },
  group: {
    marginTop: 16,
    backgroundColor: '#fff',
    borderRadius: 18,
    borderWidth: 1,
    borderColor: colors.line,
    overflow: 'hidden',
  },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    paddingHorizontal: 16,
    paddingVertical: 14,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: colors.line,
  },
  rowIcon: { width: 24, textAlign: 'center', color: colors.green, fontSize: 15, fontWeight: '800' },
  rowTitle: { color: colors.ink, fontSize: 14, fontWeight: '700' },
  rowHint: { color: colors.faint, fontSize: 10, marginTop: 3 },
  rowArrow: { color: '#b9c1ba', fontSize: 20 },
  adminButton: {
    marginTop: 14,
    backgroundColor: colors.ink,
    borderRadius: 18,
    padding: 16,
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
  },
  adminIcon: { color: colors.lime, fontSize: 18 },
  adminTitle: { color: '#fff', fontSize: 14, fontWeight: '800' },
  adminHint: { color: '#c3ccc5', fontSize: 10, marginTop: 4 },
  adminArrow: { color: colors.lime, fontSize: 20 },
  lockedCard: {
    marginTop: 14,
    backgroundColor: '#eef0eb',
    borderRadius: 18,
    padding: 16,
    borderWidth: 1,
    borderColor: '#e3e7df',
  },
  lockedTitle: { color: '#9aa29b', fontSize: 13, fontWeight: '800' },
  lockedHint: { color: '#a8b0a9', fontSize: 10, marginTop: 5, lineHeight: 15 },
  logout: { marginTop: 22, alignItems: 'center', paddingVertical: 12 },
  logoutText: { color: colors.danger, fontSize: 12, fontWeight: '700' },
  footer: { textAlign: 'center', color: colors.faint, fontSize: 9, marginTop: 8 },
});
