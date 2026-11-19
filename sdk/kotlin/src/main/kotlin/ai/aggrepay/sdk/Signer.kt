package ai.aggrepay.sdk

import java.nio.charset.StandardCharsets
import java.security.MessageDigest
import javax.crypto.Mac
import javax.crypto.spec.SecretKeySpec

object Signer {
    fun bodySha256Hex(body: ByteArray): String {
        val digest = MessageDigest.getInstance("SHA-256").digest(body)
        return digest.joinToString("") { "%02x".format(it) }
    }

    fun bodySha256Hex(body: String): String =
        bodySha256Hex(body.toByteArray(StandardCharsets.UTF_8))

    /** METHOD\nPATH\nTIMESTAMP\nNONCE\nSHA256(body) */
    fun requestCanonical(
        method: String,
        path: String,
        timestamp: String,
        nonce: String,
        body: ByteArray,
    ): String = buildString {
        append(method.uppercase())
        append('\n')
        append(path)
        append('\n')
        append(timestamp)
        append('\n')
        append(nonce)
        append('\n')
        append(bodySha256Hex(body))
    }

    fun sign(secret: String, canonical: String): String {
        val mac = Mac.getInstance("HmacSHA256")
        mac.init(SecretKeySpec(secret.toByteArray(StandardCharsets.UTF_8), "HmacSHA256"))
        return mac.doFinal(canonical.toByteArray(StandardCharsets.UTF_8))
            .joinToString("") { "%02x".format(it) }
    }

    /** Path всегда /merchant/callback. */
    fun callbackCanonical(timestamp: String, body: ByteArray): String =
        "POST\n/merchant/callback\n$timestamp\n${bodySha256Hex(body)}"

    fun verifyCallback(
        secret: String,
        timestamp: String,
        body: ByteArray,
        signature: String,
    ): Boolean {
        if (secret.isEmpty() || timestamp.isEmpty() || signature.isEmpty()) return false
        val expected = sign(secret, callbackCanonical(timestamp, body))
        return expected.equals(signature, ignoreCase = true)
    }
}
