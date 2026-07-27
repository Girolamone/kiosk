import type { ReactNode } from 'react'
import {
  ActivityIndicator,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
  type TextInputProps,
} from 'react-native'
import { colors, radius, spacing } from './theme'

export function Button({
  label,
  onPress,
  busy,
  variant = 'primary',
  disabled,
}: {
  label: string
  onPress: () => void
  busy?: boolean
  variant?: 'primary' | 'secondary'
  disabled?: boolean
}) {
  const off = busy || disabled
  return (
    <Pressable
      onPress={onPress}
      disabled={off}
      // A tap with no feedback feels broken on a phone in a way it does not
      // on a mouse, so the press state is visible.
      style={({ pressed }) => [
        styles.button,
        variant === 'primary' ? styles.buttonPrimary : styles.buttonSecondary,
        pressed && !off && styles.buttonPressed,
        off && styles.buttonOff,
      ]}
      accessibilityRole="button"
      accessibilityState={{ disabled: !!off, busy: !!busy }}
    >
      {busy && (
        <ActivityIndicator
          size="small"
          color={variant === 'primary' ? '#fff' : colors.ink}
          style={{ marginRight: spacing.sm }}
        />
      )}
      <Text style={variant === 'primary' ? styles.buttonTextPrimary : styles.buttonTextSecondary}>
        {label}
      </Text>
    </Pressable>
  )
}

export function Field({
  label,
  hint,
  ...rest
}: TextInputProps & { label: string; hint?: string }) {
  return (
    <View style={{ marginBottom: spacing.md }}>
      <Text style={styles.label}>{label}</Text>
      <TextInput
        style={styles.input}
        placeholderTextColor={colors.muted}
        // The system keyboard follows the app's palette rather than
        // flashing white over a warm page.
        keyboardAppearance="light"
        {...rest}
      />
      {hint ? <Text style={styles.hint}>{hint}</Text> : null}
    </View>
  )
}

export function Notice({ children, tone = 'info' }: { children: ReactNode; tone?: 'info' | 'error' }) {
  if (!children) return null
  return (
    <View style={[styles.notice, tone === 'error' && styles.noticeError]}>
      <Text style={[styles.noticeText, tone === 'error' && styles.noticeTextError]}>
        {children}
      </Text>
    </View>
  )
}

export function Centered({ children }: { children: ReactNode }) {
  return <View style={styles.centered}>{children}</View>
}

const styles = StyleSheet.create({
  button: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: 14,
    paddingHorizontal: spacing.lg,
    borderRadius: radius.sm,
  },
  buttonPrimary: { backgroundColor: colors.accent },
  buttonSecondary: { backgroundColor: colors.raised, borderWidth: 1, borderColor: colors.line },
  buttonPressed: { opacity: 0.85 },
  buttonOff: { opacity: 0.5 },
  buttonTextPrimary: { color: '#fff', fontSize: 15, fontWeight: '600' },
  buttonTextSecondary: { color: colors.ink, fontSize: 15, fontWeight: '600' },

  label: { color: colors.ink, fontSize: 13, fontWeight: '600', marginBottom: 6 },
  input: {
    backgroundColor: colors.raised,
    borderWidth: 1,
    borderColor: colors.line,
    borderRadius: radius.sm,
    paddingHorizontal: 12,
    paddingVertical: 11,
    fontSize: 16,
    color: colors.ink,
  },
  hint: { color: colors.muted, fontSize: 12, marginTop: 5 },

  notice: {
    backgroundColor: colors.accentSoft,
    borderRadius: radius.sm,
    padding: 12,
    marginBottom: spacing.md,
  },
  noticeError: { backgroundColor: '#fbeae9' },
  noticeText: { color: colors.accent, fontSize: 14 },
  noticeTextError: { color: colors.danger },

  centered: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: spacing.lg },
})
