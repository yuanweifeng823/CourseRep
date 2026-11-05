import { Modal, ScrollView, StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { colors } from './theme';

type Props = {
  visible: boolean;
  departments: string[];
  current?: string;
  title?: string;
  hint?: string;
  onSelect: (department: string) => void;
  onClose: () => void;
};

export function DepartmentPicker({ visible, departments, current, title, hint, onSelect, onClose }: Props) {
  return (
    <Modal visible={visible} animationType="slide" transparent onRequestClose={onClose}>
      <View style={styles.backdrop}>
        <View style={styles.sheet}>
          <Text style={styles.title}>{title ?? '选择专业'}</Text>
          <Text style={styles.hint}>{hint ?? '首页与课程页会按专业展示内容，之后可以在「我的」里修改。'}</Text>
          <ScrollView style={{ maxHeight: 380 }}>
            {departments.map((item) => (
              <TouchableOpacity
                key={item}
                style={[styles.row, current === item && styles.rowActive]}
                onPress={() => onSelect(item)}
              >
                <Text style={[styles.rowText, current === item && styles.rowTextActive]}>{item}</Text>
                {current === item ? <Text style={styles.check}>✓</Text> : null}
              </TouchableOpacity>
            ))}
            {departments.length === 0 ? <Text style={styles.empty}>暂时没有可选专业</Text> : null}
          </ScrollView>
          <TouchableOpacity style={styles.close} onPress={onClose}>
            <Text style={styles.closeText}>取消</Text>
          </TouchableOpacity>
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, backgroundColor: '#00000055', justifyContent: 'flex-end' },
  sheet: {
    backgroundColor: colors.paper,
    borderTopLeftRadius: 24,
    borderTopRightRadius: 24,
    padding: 22,
    paddingBottom: 30,
  },
  title: { color: colors.ink, fontSize: 19, fontWeight: '800' },
  hint: { color: colors.muted, fontSize: 12, lineHeight: 18, marginTop: 6, marginBottom: 14 },
  row: {
    backgroundColor: colors.card,
    borderWidth: 1,
    borderColor: colors.line,
    borderRadius: 14,
    paddingVertical: 15,
    paddingHorizontal: 16,
    marginBottom: 8,
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  rowActive: { backgroundColor: colors.ink, borderColor: colors.ink },
  rowText: { color: colors.ink, fontSize: 15, fontWeight: '700' },
  rowTextActive: { color: '#fff' },
  check: { color: colors.lime, fontSize: 16, fontWeight: '800' },
  empty: { color: colors.muted, textAlign: 'center', paddingVertical: 18 },
  close: { alignItems: 'center', paddingVertical: 14 },
  closeText: { color: colors.muted, fontSize: 13 },
});
