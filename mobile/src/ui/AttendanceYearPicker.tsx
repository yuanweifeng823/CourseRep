import { DepartmentPicker } from './DepartmentPicker';

type Props = {
  visible: boolean;
  value: string;
  onChange: (value: string) => void;
  onClose: () => void;
  allowAll?: boolean;
};

const unspecified = '年份未注明';
const all = '全部年份';

export function AttendanceYearPicker({ visible, value, onChange, onClose, allowAll }: Props) {
  const currentYear = new Date().getFullYear();
  const years = Array.from({ length: currentYear - 1990 + 1 }, (_, index) => String(currentYear - index));
  return (
    <DepartmentPicker visible={visible} title="选择修读年份"
      hint={allowAll ? '可单独查看年份未注明的经验。' : '记不清修读时间时，可以选择年份未注明。'}
      departments={[...(allowAll ? [all] : []), unspecified, ...years]}
      current={value === 'unspecified' ? unspecified : value || all}
      onSelect={(selected) => onChange(selected === unspecified ? 'unspecified' : selected === all ? '' : selected)}
      onClose={onClose} />
  );
}
