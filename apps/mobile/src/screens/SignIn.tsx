import { useState } from 'react'
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, Text, View } from 'react-native'
import { useMutation } from 'urql'
import { CreateAccessTokenDocument } from '@kiosk/shared'
import { Button, Field, Notice } from '../components'
import { colors, spacing } from '../theme'

export function SignIn({ onSignedIn }: { onSignedIn: (token: string) => Promise<void> }) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [{ fetching, error }, createToken] = useMutation(CreateAccessTokenDocument)

  async function submit() {
    const result = await createToken({ email: email.trim(), password })
    const token = result.data?.createAccessToken.token
    if (token) await onSignedIn(token)
  }

  return (
    <KeyboardAvoidingView
      style={{ flex: 1 }}
      // Without this the keyboard covers the password field on iOS and the
      // sign-in button becomes unreachable.
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
    >
      <ScrollView contentContainerStyle={styles.page} keyboardShouldPersistTaps="handled">
        <Text style={styles.title}>Kiosk</Text>
        <Text style={styles.subtitle}>Sign in to manage your shop.</Text>

        <View style={{ marginTop: spacing.xl }}>
          <Field
            label="Email"
            value={email}
            onChangeText={setEmail}
            autoCapitalize="none"
            autoCorrect={false}
            keyboardType="email-address"
            textContentType="username"
            placeholder="you@example.com"
          />
          <Field
            label="Password"
            value={password}
            onChangeText={setPassword}
            secureTextEntry
            textContentType="password"
            onSubmitEditing={submit}
            returnKeyType="go"
          />

          <Notice tone="error">
            {error ? error.message.replace(/^\[GraphQL\]\s*/, '') : null}
          </Notice>

          <Button label="Sign in" onPress={submit} busy={fetching} />
        </View>
      </ScrollView>
    </KeyboardAvoidingView>
  )
}

const styles = StyleSheet.create({
  page: { flexGrow: 1, justifyContent: 'center', padding: spacing.lg, backgroundColor: colors.paper },
  title: { fontSize: 40, color: colors.ink, fontWeight: '300' },
  subtitle: { fontSize: 15, color: colors.muted, marginTop: spacing.xs },
})
