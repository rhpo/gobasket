package life

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"math"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

const (
	// Standard sample rate for audio
	SampleRate = 44100
)

// AudioManager handles all audio operations
type AudioManager struct {
	context      *audio.Context
	sounds       map[string]*Sound
	music        map[string]*Music
	mutex        sync.RWMutex
	masterVolume float64
	musicVolume  float64
	soundVolume  float64
	currentMusic *Music
}

// Sound represents a short audio clip (sound effects)
type Sound struct {
	name    string
	data    []byte
	volume  float64
	players []*audio.Player
	mutex   sync.Mutex
}

// Music represents longer audio tracks (background music)
type Music struct {
	name   string
	data   []byte
	volume float64
	player *audio.Player
	loop   bool
	mutex  sync.Mutex
}

// AudioProps contains properties for audio configuration
type AudioProps struct {
	MasterVolume float64
	MusicVolume  float64
	SoundVolume  float64
}

// NewAudioManager creates a new audio manager
func NewAudioManager(props *AudioProps) *AudioManager {
	if props == nil {
		props = &AudioProps{
			MasterVolume: 1.0,
			MusicVolume:  0.7,
			SoundVolume:  0.8,
		}
	}

	return &AudioManager{
		context:      audio.NewContext(SampleRate),
		sounds:       make(map[string]*Sound),
		music:        make(map[string]*Music),
		masterVolume: props.MasterVolume,
		musicVolume:  props.MusicVolume,
		soundVolume:  props.SoundVolume,
	}
}

// LoadSound loads a sound effect from file path
func (am *AudioManager) LoadSound(name, filePath string) error {
	// This would load from regular file system
	// Implementation depends on your file structure
	return fmt.Errorf("LoadSound from file path not implemented - use LoadSoundFromFS")
}

// LoadSoundFromFS loads a sound effect from embedded filesystem
func (am *AudioManager) LoadSoundFromFS(name string, fs embed.FS, filePath string) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	data, err := fs.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read audio file %s: %w", filePath, err)
	}

	if len(data) == 0 {
		return fmt.Errorf("audio file %s is empty", filePath)
	}

	audioData, err := am.decodeAudio(data, filePath)
	if err != nil {
		return fmt.Errorf("failed to decode audio file %s: %w", filePath, err)
	}

	if len(audioData) == 0 {
		return fmt.Errorf("decoded audio data for %s is empty", filePath)
	}

	sound := &Sound{
		name:    name,
		data:    audioData,
		volume:  1.0,
		players: make([]*audio.Player, 0),
	}

	am.sounds[name] = sound
	return nil
}

// LoadMusicFromFS loads background music from embedded filesystem
func (am *AudioManager) LoadMusicFromFS(name string, fs embed.FS, filePath string) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	data, err := fs.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read music file %s: %w", filePath, err)
	}

	audioData, err := am.decodeAudio(data, filePath)
	if err != nil {
		return fmt.Errorf("failed to decode music file %s: %w", filePath, err)
	}

	music := &Music{
		name:   name,
		data:   audioData,
		volume: 1.0,
		loop:   true,
	}

	am.music[name] = music
	return nil
}

// decodeAudio decodes audio data based on file extension
func (am *AudioManager) decodeAudio(data []byte, filePath string) ([]byte, error) {
	reader := bytes.NewReader(data)

	// Determine format by file extension
	if len(filePath) < 4 {
		return nil, fmt.Errorf("invalid file path: %s", filePath)
	}

	ext := filePath[len(filePath)-4:]

	var stream io.Reader
	var err error

	switch ext {
	case ".mp3":
		stream, err = mp3.DecodeWithSampleRate(SampleRate, reader)
	case ".wav":
		stream, err = wav.DecodeWithSampleRate(SampleRate, reader)
	case ".ogg":
		stream, err = vorbis.DecodeWithSampleRate(SampleRate, reader)
	default:
		return nil, fmt.Errorf("unsupported audio format: %s", ext)
	}

	if err != nil {
		return nil, err
	}

	// Read all data from the stream
	return io.ReadAll(stream)
}

// PlaySound plays a sound effect
func (am *AudioManager) PlaySound(name string) error {
	return am.PlaySoundWithVolume(name, 1.0)
}

// createPlayerFromData creates an audio player from decoded audio data
func (am *AudioManager) createPlayerFromData(data []byte) (*audio.Player, error) {
	reader := bytes.NewReader(data)
	return am.context.NewPlayer(reader)
}

// PlaySoundWithVolume plays a sound effect with specific volume
func (am *AudioManager) PlaySoundWithVolume(name string, volume float64) error {
	am.mutex.RLock()
	sound, exists := am.sounds[name]
	am.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("sound %s not found", name)
	}

	if len(sound.data) == 0 {
		return fmt.Errorf("sound %s has no data", name)
	}

	sound.mutex.Lock()
	defer sound.mutex.Unlock()

	// Create a new player for this sound instance
	player, err := am.createPlayerFromData(sound.data)
	if err != nil {
		return fmt.Errorf("failed to create audio player for %s: %w", name, err)
	}

	// Calculate final volume
	finalVolume := am.masterVolume * am.soundVolume * sound.volume * volume
	if finalVolume <= 0 {
		return fmt.Errorf("calculated volume is 0 for sound %s (master: %f, sound: %f, individual: %f, requested: %f)",
			name, am.masterVolume, am.soundVolume, sound.volume, volume)
	}

	player.SetVolume(finalVolume)

	// Clean up finished players
	am.cleanupSoundPlayers(sound)

	// Add to active players
	sound.players = append(sound.players, player)

	// Play the sound
	player.Play()

	return nil
}

// PlayMusic plays background music
func (am *AudioManager) PlayMusic(name string) error {
	return am.PlayMusicWithOptions(name, true, 1.0)
}

// PlayMusicWithOptions plays background music with specific options
func (am *AudioManager) PlayMusicWithOptions(name string, loop bool, volume float64) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	music, exists := am.music[name]
	if !exists {
		return fmt.Errorf("music %s not found", name)
	}

	// Stop current music if playing
	if am.currentMusic != nil {
		am.stopCurrentMusic()
	}

	music.mutex.Lock()
	defer music.mutex.Unlock()

	// Create new player
	player, err := am.createPlayerFromData(music.data)
	if err != nil {
		return fmt.Errorf("failed to create music player for %s: %w", name, err)
	}

	// Configure player
	finalVolume := am.masterVolume * am.musicVolume * music.volume * volume
	player.SetVolume(finalVolume)

	music.player = player
	music.loop = loop
	am.currentMusic = music

	// Start playing
	player.Play()

	// Handle looping in a goroutine
	if loop {
		go am.handleMusicLoop(music)
	}

	return nil
}

// handleMusicLoop handles music looping
func (am *AudioManager) handleMusicLoop(music *Music) {
	for {
		time.Sleep(100 * time.Millisecond)

		music.mutex.Lock()
		if music.player == nil || !music.loop {
			music.mutex.Unlock()
			break
		}

		if !music.player.IsPlaying() {
			// Restart the music
			music.player.Rewind()
			music.player.Play()
		}
		music.mutex.Unlock()
	}
}

// StopMusic stops the currently playing music
func (am *AudioManager) StopMusic() {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	am.stopCurrentMusic()
}

// stopCurrentMusic stops current music (internal, assumes mutex is locked)
func (am *AudioManager) stopCurrentMusic() {
	if am.currentMusic != nil {
		am.currentMusic.mutex.Lock()
		if am.currentMusic.player != nil {
			am.currentMusic.player.Close()
			am.currentMusic.player = nil
		}
		am.currentMusic.loop = false
		am.currentMusic.mutex.Unlock()
		am.currentMusic = nil
	}
}

// PauseMusic pauses the currently playing music
func (am *AudioManager) PauseMusic() {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	if am.currentMusic != nil {
		am.currentMusic.mutex.Lock()
		if am.currentMusic.player != nil {
			am.currentMusic.player.Pause()
		}
		am.currentMusic.mutex.Unlock()
	}
}

// ResumeMusic resumes the currently paused music
func (am *AudioManager) ResumeMusic() {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	if am.currentMusic != nil {
		am.currentMusic.mutex.Lock()
		if am.currentMusic.player != nil {
			am.currentMusic.player.Play()
		}
		am.currentMusic.mutex.Unlock()
	}
}

// SetMasterVolume sets the master volume (0.0 to 1.0)
func (am *AudioManager) SetMasterVolume(volume float64) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	am.masterVolume = clampVolume(volume)
	am.updateAllVolumes()
}

// SetMusicVolume sets the music volume (0.0 to 1.0)
func (am *AudioManager) SetMusicVolume(volume float64) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	am.musicVolume = clampVolume(volume)

	if am.currentMusic != nil {
		am.currentMusic.mutex.Lock()
		if am.currentMusic.player != nil {
			finalVolume := am.masterVolume * am.musicVolume * am.currentMusic.volume
			am.currentMusic.player.SetVolume(finalVolume)
		}
		am.currentMusic.mutex.Unlock()
	}
}

// SetSoundVolume sets the sound effects volume (0.0 to 1.0)
func (am *AudioManager) SetSoundVolume(volume float64) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	am.soundVolume = clampVolume(volume)
}

// SetSoundVolumeByName sets the volume for a specific sound
func (am *AudioManager) SetSoundVolumeByName(name string, volume float64) error {
	am.mutex.RLock()
	sound, exists := am.sounds[name]
	am.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("sound %s not found", name)
	}

	sound.mutex.Lock()
	sound.volume = clampVolume(volume)
	sound.mutex.Unlock()

	return nil
}

// cleanupSoundPlayers removes finished sound players
func (am *AudioManager) cleanupSoundPlayers(sound *Sound) {
	activePlayers := make([]*audio.Player, 0)

	for _, player := range sound.players {
		if player.IsPlaying() {
			activePlayers = append(activePlayers, player)
		} else {
			player.Close()
		}
	}

	sound.players = activePlayers
}

// updateAllVolumes updates volumes for all active audio
func (am *AudioManager) updateAllVolumes() {
	// Update current music volume
	if am.currentMusic != nil {
		am.currentMusic.mutex.Lock()
		if am.currentMusic.player != nil {
			finalVolume := am.masterVolume * am.musicVolume * am.currentMusic.volume
			am.currentMusic.player.SetVolume(finalVolume)
		}
		am.currentMusic.mutex.Unlock()
	}

	// Note: Sound effects volumes will be updated when they're played next
	// Active sound players keep their volume until they finish
}

// GetMasterVolume returns the current master volume
func (am *AudioManager) GetMasterVolume() float64 {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	return am.masterVolume
}

// GetMusicVolume returns the current music volume
func (am *AudioManager) GetMusicVolume() float64 {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	return am.musicVolume
}

// GetSoundVolume returns the current sound effects volume
func (am *AudioManager) GetSoundVolume() float64 {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	return am.soundVolume
}

// IsMusicPlaying returns true if music is currently playing
func (am *AudioManager) IsMusicPlaying() bool {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	if am.currentMusic == nil {
		return false
	}

	am.currentMusic.mutex.Lock()
	defer am.currentMusic.mutex.Unlock()

	return am.currentMusic.player != nil && am.currentMusic.player.IsPlaying()
}

// GetSoundNames returns all loaded sound names (for debugging)
func (am *AudioManager) GetSoundNames() []string {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	names := make([]string, 0, len(am.sounds))
	for name := range am.sounds {
		names = append(names, name)
	}
	return names
}

// GetSoundInfo returns debug info about a sound
func (am *AudioManager) GetSoundInfo(name string) (bool, int, error) {
	am.mutex.RLock()
	sound, exists := am.sounds[name]
	am.mutex.RUnlock()

	if !exists {
		return false, 0, fmt.Errorf("sound %s not found", name)
	}

	return true, len(sound.data), nil
}

// CreateTestTone creates a simple test tone for debugging
func (am *AudioManager) CreateTestTone(name string, frequency float64, duration time.Duration) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Generate a simple sine wave
	samples := int(float64(SampleRate) * duration.Seconds())
	data := make([]byte, samples*4) // 16-bit stereo

	for i := 0; i < samples; i++ {
		// Generate sine wave
		t := float64(i) / float64(SampleRate)
		sample := int16(32767 * 0.1 * math.Sin(2*math.Pi*frequency*t)) // Low volume

		// Convert to bytes (little endian, stereo)
		data[i*4] = byte(sample)
		data[i*4+1] = byte(sample >> 8)
		data[i*4+2] = byte(sample)
		data[i*4+3] = byte(sample >> 8)
	}

	sound := &Sound{
		name:    name,
		data:    data,
		volume:  1.0,
		players: make([]*audio.Player, 0),
	}

	am.sounds[name] = sound
}

// Update should be called every frame to update the audio system
func (am *AudioManager) Update() {
	// Ebiten's audio context doesn't need explicit updates in newer versions
	// but we can use this for cleanup and maintenance
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	// Clean up finished sound players periodically
	for _, sound := range am.sounds {
		sound.mutex.Lock()
		am.cleanupSoundPlayers(sound)
		sound.mutex.Unlock()
	}
}

// Cleanup cleans up all audio resources
func (am *AudioManager) Cleanup() {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Stop and cleanup music
	am.stopCurrentMusic()

	// Cleanup all sounds
	for _, sound := range am.sounds {
		sound.mutex.Lock()
		for _, player := range sound.players {
			player.Close()
		}
		sound.players = nil
		sound.mutex.Unlock()
	}

	am.sounds = make(map[string]*Sound)
	am.music = make(map[string]*Music)
}

// clampVolume ensures volume is between 0.0 and 1.0
func clampVolume(volume float64) float64 {
	if volume < 0.0 {
		return 0.0
	}
	if volume > 1.0 {
		return 1.0
	}
	return volume
}

// Global audio manager instance
var globalAudioManager *AudioManager

// InitAudio initializes the global audio manager
func InitAudio(props *AudioProps) {
	globalAudioManager = NewAudioManager(props)
}

// GetAudioManager returns the global audio manager
func GetAudioManager() *AudioManager {
	if globalAudioManager == nil {
		InitAudio(nil) // Initialize with defaults
	}
	return globalAudioManager
}

// Convenience functions for global audio manager

// LoadSound loads a sound using the global audio manager
func LoadSound(name string, fs embed.FS, filePath string) error {
	return GetAudioManager().LoadSoundFromFS(name, fs, filePath)
}

// LoadMusic loads music using the global audio manager
func LoadMusic(name string, fs embed.FS, filePath string) error {
	return GetAudioManager().LoadMusicFromFS(name, fs, filePath)
}

// PlaySound plays a sound using the global audio manager
func PlaySound(name string) error {
	return GetAudioManager().PlaySound(name)
}

// PlaySoundWithVolume plays a sound with volume using the global audio manager
func PlaySoundWithVolume(name string, volume float64) error {
	return GetAudioManager().PlaySoundWithVolume(name, volume)
}

// PlayMusic plays music using the global audio manager
func PlayMusic(name string) error {
	return GetAudioManager().PlayMusic(name)
}

// PlayMusicWithOptions plays music with options using the global audio manager
func PlayMusicWithOptions(name string, loop bool, volume float64) error {
	return GetAudioManager().PlayMusicWithOptions(name, loop, volume)
}

// StopMusic stops music using the global audio manager
func StopMusic() {
	GetAudioManager().StopMusic()
}

// PauseMusic pauses music using the global audio manager
func PauseMusic() {
	GetAudioManager().PauseMusic()
}

// ResumeMusic resumes music using the global audio manager
func ResumeMusic() {
	GetAudioManager().ResumeMusic()
}
