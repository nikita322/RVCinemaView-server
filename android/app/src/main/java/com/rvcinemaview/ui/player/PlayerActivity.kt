package com.rvcinemaview.ui.player

import android.os.Bundle
import android.view.KeyEvent
import android.view.View
import android.view.WindowManager
import androidx.appcompat.app.AppCompatActivity
import androidx.core.view.WindowCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.WindowInsetsControllerCompat
import androidx.lifecycle.lifecycleScope
import androidx.media3.common.MediaItem
import androidx.media3.common.PlaybackException
import androidx.media3.common.Player
import androidx.media3.exoplayer.ExoPlayer
import com.rvcinemaview.data.api.ApiClient
import com.rvcinemaview.databinding.ActivityPlayerBinding

class PlayerActivity : AppCompatActivity() {

    companion object {
        const val EXTRA_MEDIA_ID = "media_id"
        const val EXTRA_MEDIA_TITLE = "media_title"
        const val EXTRA_START_POSITION = "start_position"

        private const val SEEK_INCREMENT_MS = 10_000L
        private const val SEEK_INCREMENT_LONG_MS = 60_000L
        private const val SAVE_POSITION_INTERVAL_MS = 10_000L // Save every 10 seconds
    }

    private lateinit var binding: ActivityPlayerBinding
    private var player: ExoPlayer? = null

    private var mediaId: String? = null
    private var mediaTitle: String? = null
    private var startPosition: Long = 0L
    private var lastSavedPosition: Long = 0L

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityPlayerBinding.inflate(layoutInflater)
        setContentView(binding.root)

        mediaId = intent.getStringExtra(EXTRA_MEDIA_ID)
        mediaTitle = intent.getStringExtra(EXTRA_MEDIA_TITLE)
        startPosition = intent.getLongExtra(EXTRA_START_POSITION, 0L)

        if (mediaId == null) {
            showError("No media ID provided")
            return
        }

        hideSystemUI()
        keepScreenOn()
    }

    override fun onStart() {
        super.onStart()
        initializePlayer()
    }

    override fun onPause() {
        super.onPause()
        savePlaybackPosition()
    }

    override fun onStop() {
        super.onStop()
        releasePlayer()
    }

    private fun initializePlayer() {
        if (player != null) return

        player = ExoPlayer.Builder(this)
            .build()
            .also { exoPlayer ->
                binding.playerView.player = exoPlayer

                exoPlayer.addListener(object : Player.Listener {
                    override fun onPlaybackStateChanged(playbackState: Int) {
                        updateLoadingState(playbackState)
                    }

                    override fun onPlayerError(error: PlaybackException) {
                        showError("Playback error: ${error.localizedMessage}")
                    }

                    override fun onPositionDiscontinuity(
                        oldPosition: Player.PositionInfo,
                        newPosition: Player.PositionInfo,
                        reason: Int
                    ) {
                        // Save position periodically during playback
                        maybeSavePosition()
                    }
                })

                val streamUrl = ApiClient.getStreamUrl(mediaId!!)
                val mediaItem = MediaItem.fromUri(streamUrl)

                exoPlayer.setMediaItem(mediaItem)
                exoPlayer.prepare()

                // Seek to saved position or load from server
                if (startPosition > 0) {
                    exoPlayer.seekTo(startPosition * 1000) // Convert to milliseconds
                    exoPlayer.playWhenReady = true
                } else {
                    // Load saved position from server, then start playback
                    loadSavedPosition(exoPlayer)
                }
            }
    }

    private fun releasePlayer() {
        player?.let { exoPlayer ->
            exoPlayer.release()
        }
        player = null
    }

    private fun updateLoadingState(playbackState: Int) {
        when (playbackState) {
            Player.STATE_BUFFERING -> {
                binding.loadingProgress.visibility = View.VISIBLE
                binding.errorText.visibility = View.GONE
            }
            Player.STATE_READY -> {
                binding.loadingProgress.visibility = View.GONE
                binding.errorText.visibility = View.GONE
            }
            Player.STATE_ENDED -> {
                finish()
            }
            Player.STATE_IDLE -> {
                binding.loadingProgress.visibility = View.GONE
            }
        }
    }

    private fun showError(message: String) {
        binding.loadingProgress.visibility = View.GONE
        binding.errorText.text = message
        binding.errorText.visibility = View.VISIBLE
    }

    private fun hideSystemUI() {
        WindowCompat.setDecorFitsSystemWindows(window, false)
        WindowInsetsControllerCompat(window, binding.root).let { controller ->
            controller.hide(WindowInsetsCompat.Type.systemBars())
            controller.systemBarsBehavior =
                WindowInsetsControllerCompat.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE
        }
    }

    private fun keepScreenOn() {
        window.addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON)
    }

    // D-Pad and remote control support
    override fun onKeyDown(keyCode: Int, event: KeyEvent?): Boolean {
        val player = this.player ?: return super.onKeyDown(keyCode, event)

        return when (keyCode) {
            // Play/Pause
            KeyEvent.KEYCODE_MEDIA_PLAY_PAUSE,
            KeyEvent.KEYCODE_DPAD_CENTER -> {
                if (player.isPlaying) {
                    player.pause()
                } else {
                    player.play()
                }
                true
            }

            KeyEvent.KEYCODE_MEDIA_PLAY -> {
                player.play()
                true
            }

            KeyEvent.KEYCODE_MEDIA_PAUSE -> {
                player.pause()
                true
            }

            // Seek forward
            KeyEvent.KEYCODE_DPAD_RIGHT,
            KeyEvent.KEYCODE_MEDIA_FAST_FORWARD -> {
                val increment = if (keyCode == KeyEvent.KEYCODE_MEDIA_FAST_FORWARD) {
                    SEEK_INCREMENT_LONG_MS
                } else {
                    SEEK_INCREMENT_MS
                }
                player.seekTo(player.currentPosition + increment)
                true
            }

            // Seek backward
            KeyEvent.KEYCODE_DPAD_LEFT,
            KeyEvent.KEYCODE_MEDIA_REWIND -> {
                val increment = if (keyCode == KeyEvent.KEYCODE_MEDIA_REWIND) {
                    SEEK_INCREMENT_LONG_MS
                } else {
                    SEEK_INCREMENT_MS
                }
                player.seekTo(maxOf(0, player.currentPosition - increment))
                true
            }

            // Stop and exit
            KeyEvent.KEYCODE_MEDIA_STOP -> {
                finish()
                true
            }

            // Back button
            KeyEvent.KEYCODE_BACK -> {
                finish()
                true
            }

            else -> super.onKeyDown(keyCode, event)
        }
    }

    private fun loadSavedPosition(exoPlayer: ExoPlayer) {
        val id = mediaId ?: return
        PlaybackHelper.loadSavedPosition(lifecycleScope, id, exoPlayer) { position ->
            startPosition = position
        }
        exoPlayer.playWhenReady = true
    }

    private fun maybeSavePosition() {
        val exoPlayer = player ?: return
        val currentPositionMs = exoPlayer.currentPosition
        val currentPositionSec = currentPositionMs / 1000

        // Only save if position changed significantly (10 seconds)
        if (kotlin.math.abs(currentPositionSec - lastSavedPosition) >= SAVE_POSITION_INTERVAL_MS / 1000) {
            lastSavedPosition = currentPositionSec
            savePlaybackPosition()
        }
    }

    private fun savePlaybackPosition() {
        val id = mediaId ?: return
        PlaybackHelper.savePlaybackPosition(lifecycleScope, id, player)
    }
}
