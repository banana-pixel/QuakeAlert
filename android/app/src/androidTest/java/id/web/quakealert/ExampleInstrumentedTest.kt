// Copyright (c) 2026 QuakeAlert contributors.
// SPDX-License-Identifier: GPL-3.0-or-later
// See android/LICENSE-EXCEPTION for the Google Play Services linking permission.

package id.web.quakealert

import androidx.test.platform.app.InstrumentationRegistry
import androidx.test.ext.junit.runners.AndroidJUnit4

import org.junit.Test
import org.junit.runner.RunWith

import org.junit.Assert.*

/**
 * Instrumented test, which will execute on an Android device.
 *
 * See [testing documentation](http://d.android.com/tools/testing).
 */
@RunWith(AndroidJUnit4::class)
class ExampleInstrumentedTest {
    @Test
    fun useAppContext() {
        // Context of the app under test.
        val appContext = InstrumentationRegistry.getInstrumentation().targetContext
        assertEquals("id.web.quakealert", appContext.packageName)
    }
}