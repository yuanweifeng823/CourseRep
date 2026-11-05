export const colors = {
  green: '#315f4a',
  lime: '#d9ed9c',
  ink: '#202722',
  paper: '#f5f6f2',
  card: '#ffffff',
  line: '#e9ece6',
  muted: '#78817a',
  faint: '#9aa29b',
  danger: '#a34e3c',
};

export const layout = {
  gutter: 20,
  radius: 16,
};

export function statusLabel(status?: string) {
  return (
    {
      verified: '已验证',
      alumni_readonly: '校友只读',
      unverified: '未验证',
      submitted: '审核中',
      under_review: '复核中',
      published: '已发布',
      rejected: '未通过',
      removed: '已下架',
      disabled: '已停用',
    } as Record<string, string>
  )[status ?? ''] ?? '未知';
}
