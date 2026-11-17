import { useLayoutEffect } from 'react';
import { Alert } from 'react-native';

export type LeaveState = boolean | 'busy';

export function useUnsavedChanges(dirty: LeaveState, onChange?: (dirty: LeaveState) => void) {
  useLayoutEffect(() => {
    if (!dirty || !onChange) return;
    onChange(dirty);
    return () => onChange(false);
  }, [dirty, onChange]);
}

export function confirmDiscard(dirty: LeaveState, leave: () => void) {
  if (dirty === 'busy') {
    Alert.alert('正在处理', '请等当前操作完成后再离开。');
    return;
  }
  if (!dirty) {
    leave();
    return;
  }
  Alert.alert('放弃未提交的内容？', '离开后，本次修改将丢失。', [
    { text: '继续编辑', style: 'cancel' },
    { text: '放弃并离开', style: 'destructive', onPress: leave },
  ]);
}
