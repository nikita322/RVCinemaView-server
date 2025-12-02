package com.rvcinemaview.ui.settings

import android.content.Intent
import android.os.Bundle
import androidx.appcompat.app.AlertDialog
import androidx.appcompat.app.AppCompatActivity
import androidx.lifecycle.lifecycleScope
import com.rvcinemaview.BuildConfig
import com.rvcinemaview.R
import com.rvcinemaview.data.repository.LibraryRepository
import com.rvcinemaview.data.repository.ServerPreferences
import com.rvcinemaview.databinding.ActivitySettingsBinding
import com.rvcinemaview.ui.connect.ConnectActivity
import kotlinx.coroutines.flow.firstOrNull
import kotlinx.coroutines.launch

class SettingsActivity : AppCompatActivity() {

    private lateinit var binding: ActivitySettingsBinding
    private lateinit var serverPreferences: ServerPreferences

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivitySettingsBinding.inflate(layoutInflater)
        setContentView(binding.root)

        serverPreferences = ServerPreferences(this)

        setupToolbar()
        setupViews()
        loadSettings()
    }

    private fun setupToolbar() {
        binding.toolbar.setNavigationOnClickListener {
            finish()
        }
    }

    private fun setupViews() {
        binding.resetButton.setOnClickListener {
            showResetConfirmation()
        }

        // Set version
        binding.versionText.text = BuildConfig.VERSION_NAME
    }

    private fun loadSettings() {
        lifecycleScope.launch {
            val serverAddress = serverPreferences.serverAddress.firstOrNull()
            binding.serverAddressText.text = serverAddress ?: "-"
        }
    }

    private fun showResetConfirmation() {
        AlertDialog.Builder(this, R.style.Theme_RVCinemaView_Dialog)
            .setTitle(R.string.settings_reset_confirm_title)
            .setMessage(R.string.settings_reset_confirm_message)
            .setPositiveButton(R.string.settings_reset_confirm) { _, _ ->
                resetSettings()
            }
            .setNegativeButton(R.string.cancel, null)
            .show()
    }

    private fun resetSettings() {
        lifecycleScope.launch {
            // Clear library cache
            LibraryRepository.getInstance().clear()

            // Clear server address
            serverPreferences.clearServerAddress()

            // Navigate to connect screen
            val intent = Intent(this@SettingsActivity, ConnectActivity::class.java)
            intent.flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TASK
            startActivity(intent)
            finish()
        }
    }
}
