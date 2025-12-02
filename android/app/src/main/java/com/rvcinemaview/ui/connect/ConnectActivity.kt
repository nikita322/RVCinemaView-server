package com.rvcinemaview.ui.connect

import android.content.Intent
import android.os.Bundle
import android.view.KeyEvent
import android.view.View
import android.view.inputmethod.EditorInfo
import androidx.appcompat.app.AppCompatActivity
import androidx.lifecycle.lifecycleScope
import com.rvcinemaview.R
import com.rvcinemaview.data.api.ApiClient
import com.rvcinemaview.data.repository.ServerPreferences
import com.rvcinemaview.databinding.ActivityConnectBinding
import com.rvcinemaview.ui.browse.UnifiedBrowseActivity
import kotlinx.coroutines.flow.firstOrNull
import kotlinx.coroutines.launch

class ConnectActivity : AppCompatActivity() {

    private lateinit var binding: ActivityConnectBinding
    private lateinit var serverPreferences: ServerPreferences
    private var isAutoConnecting = false

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityConnectBinding.inflate(layoutInflater)
        setContentView(binding.root)

        serverPreferences = ServerPreferences(this)

        setupViews()
        tryAutoConnect()
    }

    private fun setupViews() {
        binding.connectButton.setOnClickListener {
            connect()
        }

        binding.serverAddressInput.setOnEditorActionListener { _, actionId, event ->
            if (actionId == EditorInfo.IME_ACTION_DONE ||
                (event?.keyCode == KeyEvent.KEYCODE_ENTER && event.action == KeyEvent.ACTION_DOWN)) {
                connect()
                true
            } else {
                false
            }
        }

        // D-Pad support for TV
        binding.serverAddressInput.setOnKeyListener { _, keyCode, event ->
            if (keyCode == KeyEvent.KEYCODE_DPAD_DOWN && event.action == KeyEvent.ACTION_DOWN) {
                binding.connectButton.requestFocus()
                true
            } else {
                false
            }
        }
    }

    private fun tryAutoConnect() {
        lifecycleScope.launch {
            val savedAddress = serverPreferences.serverAddress.firstOrNull()
            if (!savedAddress.isNullOrEmpty()) {
                // Hide UI and show only loading during auto-connect
                isAutoConnecting = true
                setAutoConnectMode(true)
                binding.serverAddressInput.setText(savedAddress)
                autoConnect(savedAddress)
            } else {
                // No saved address - show connection UI
                showConnectionUI()
            }
        }
    }

    private fun autoConnect(address: String) {
        lifecycleScope.launch {
            try {
                ApiClient.init(address)
                val health = ApiClient.getApi().health()

                if (health.status == "ok") {
                    startBrowseActivity()
                } else {
                    // Server responded but status is not ok
                    onAutoConnectFailed(getString(R.string.error_invalid_server_status))
                }
            } catch (e: Exception) {
                // Connection failed - show UI with error
                onAutoConnectFailed(getString(R.string.error_connection_failed, e.localizedMessage))
            }
        }
    }

    private fun onAutoConnectFailed(errorMessage: String) {
        isAutoConnecting = false
        setAutoConnectMode(false)
        showConnectionUI()
        showError(errorMessage)
    }

    private fun setAutoConnectMode(autoConnecting: Boolean) {
        // During auto-connect: hide all UI except centered progress
        binding.logoView.visibility = if (autoConnecting) View.VISIBLE else View.VISIBLE
        binding.titleText.visibility = if (autoConnecting) View.VISIBLE else View.VISIBLE
        binding.subtitleText.visibility = if (autoConnecting) View.GONE else View.VISIBLE
        binding.serverAddressInputLayout.visibility = if (autoConnecting) View.GONE else View.VISIBLE
        binding.connectButton.visibility = if (autoConnecting) View.GONE else View.VISIBLE
        binding.progressBar.visibility = if (autoConnecting) View.VISIBLE else View.GONE
        binding.statusText.visibility = View.GONE

        if (autoConnecting) {
            binding.subtitleText.text = getString(R.string.connecting)
        } else {
            binding.subtitleText.text = getString(R.string.connect_title)
        }
    }

    private fun showConnectionUI() {
        binding.logoView.visibility = View.VISIBLE
        binding.titleText.visibility = View.VISIBLE
        binding.subtitleText.visibility = View.VISIBLE
        binding.subtitleText.text = getString(R.string.connect_title)
        binding.serverAddressInputLayout.visibility = View.VISIBLE
        binding.connectButton.visibility = View.VISIBLE
        binding.progressBar.visibility = View.GONE
    }

    private fun connect() {
        val address = binding.serverAddressInput.text.toString().trim()

        if (address.isEmpty()) {
            showError(getString(R.string.error_enter_server_address))
            return
        }

        setLoading(true)

        lifecycleScope.launch {
            try {
                ApiClient.init(address)
                val health = ApiClient.getApi().health()

                if (health.status == "ok") {
                    serverPreferences.saveServerAddress(address)
                    startBrowseActivity()
                } else {
                    showError(getString(R.string.error_invalid_server_status))
                }
            } catch (e: Exception) {
                showError(getString(R.string.error_connection_failed, e.localizedMessage))
            } finally {
                setLoading(false)
            }
        }
    }

    private fun setLoading(loading: Boolean) {
        binding.connectButton.visibility = if (loading) View.INVISIBLE else View.VISIBLE
        binding.progressBar.visibility = if (loading) View.VISIBLE else View.GONE
        binding.serverAddressInput.isEnabled = !loading
        binding.statusText.visibility = View.GONE
    }

    private fun showError(message: String) {
        binding.statusText.text = message
        binding.statusText.visibility = View.VISIBLE
    }

    private fun startBrowseActivity() {
        startActivity(Intent(this, UnifiedBrowseActivity::class.java))
        finish()
    }
}
