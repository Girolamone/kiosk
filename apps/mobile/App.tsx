import { SafeAreaView, StatusBar, StyleSheet, Text, View } from 'react-native'
import { Provider as UrqlProvider } from 'urql'
import { client } from './src/api'
import { useStoredSession } from './src/session'
import { colors } from './src/theme'
import { SignIn } from './src/screens/SignIn'
import { Shop } from './src/screens/Shop'

export default function App() {
  const { token, loading, signIn, signOut } = useStoredSession()

  return (
    <UrqlProvider value={client}>
      <StatusBar barStyle="dark-content" backgroundColor={colors.paper} />
      <SafeAreaView style={styles.safe}>
        {loading ? (
          // Reading the keychain is fast but not instant, and flashing the
          // sign-in screen at someone who is already signed in looks broken.
          <View style={styles.blank}>
            <Text style={styles.muted}>Kiosk</Text>
          </View>
        ) : token ? (
          // Keyed on the token so signing out drops all screen state rather
          // than leaving the next person looking at the last one's shop.
          <Shop key={token} onSignOut={signOut} />
        ) : (
          <SignIn onSignedIn={signIn} />
        )}
      </SafeAreaView>
    </UrqlProvider>
  )
}

const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: colors.paper },
  blank: { flex: 1, alignItems: 'center', justifyContent: 'center' },
  muted: { color: colors.muted, fontSize: 18 },
})
