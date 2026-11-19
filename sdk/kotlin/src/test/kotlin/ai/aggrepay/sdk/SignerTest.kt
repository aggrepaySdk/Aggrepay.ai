package ai.aggrepay.sdk

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

class SignerTest {
    @Test
    fun requestCanonicalStable() {
        val body = """{"a":1}""".toByteArray()
        val c = Signer.requestCanonical("post", "/api/v1/invoice/create", "1700000000", "abc", body)
        assertTrue(c.startsWith("POST\n/api/v1/invoice/create\n1700000000\nabc\n"))
        assertEquals(64, c.substringAfterLast('\n').length)
    }

    @Test
    fun verifyCallbackRoundtrip() {
        val body = """{"invoice_id":"inv_1"}""".toByteArray()
        val ts = "1700000000"
        val secret = "test-secret"
        val sig = Signer.sign(secret, Signer.callbackCanonical(ts, body))
        assertTrue(Signer.verifyCallback(secret, ts, body, sig))
        assertFalse(Signer.verifyCallback(secret, ts, body, "deadbeef"))
        assertFalse(Signer.verifyCallback("", ts, body, sig))
    }
}
