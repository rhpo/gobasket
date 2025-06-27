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
	SampleRate = 44100
)

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

type Sound struct {
	name    string
	data    []byte
	volume  float64
	players []*audio.Player
	mutex   sync.Mutex
}

type Music struct {
	name   string
	data   []byte
	volume float64
	player *audio.Player
	loop   bool
	mutex  sync.Mutex
}

type AudioProps struct {
	MasterVolume float64
	MusicVolume  float64
	SoundVolume  float64
}

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

func (am *AudioManager) LoadSound(name, filePath string) error {

	return fmt.Errorf("LoadSound from file path not implemented - use LoadSoundFromFS")
}

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

func (am *AudioManager) decodeAudio(data []byte, filePath string) ([]byte, error) {
	reader := bytes.NewReader(data)

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

	return io.ReadAll(stream)
}

func (am *AudioManager) PlaySound(name string) error {
	return am.PlaySoundWithVolume(name, 1.0)
}

func (am *AudioManager) createPlayerFromData(data []byte) (*audio.Player, error) {
	reader := bytes.NewReader(data)
	return am.context.NewPlayer(reader)
}

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

	player, err := am.createPlayerFromData(sound.data)
	if err != nil {
		return fmt.Errorf("failed to create audio player for %s: %w", name, err)
	}

	finalVolume := am.masterVolume * am.soundVolume * sound.volume * volume
	if finalVolume <= 0 {
		return fmt.Errorf("calculated volume is 0 for sound %s (master: %f, sound: %f, individual: %f, requested: %f)",
			name, am.masterVolume, am.soundVolume, sound.volume, volume)
	}

	player.SetVolume(finalVolume)

	am.cleanupSoundPlayers(sound)

	sound.players = append(sound.players, player)

	player.Play()

	return nil
}

func (am *AudioManager) PlayMusic(name string) error {
	return am.PlayMusicWithOptions(name, true, 1.0)
}

func (am *AudioManager) PlayMusicWithOptions(name string, loop bool, volume float64) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	music, exists := am.music[name]
	if !exists {
		return fmt.Errorf("music %s not found", name)
	}

	if am.currentMusic != nil {
		am.stopCurrentMusic()
	}

	music.mutex.Lock()
	defer music.mutex.Unlock()

	player, err := am.createPlayerFromData(music.data)
	if err != nil {
		return fmt.Errorf("failed to create music player for %s: %w", name, err)
	}

	finalVolume := am.masterVolume * am.musicVolume * music.volume * volume
	player.SetVolume(finalVolume)

	music.player = player
	music.loop = loop
	am.currentMusic = music

	player.Play()

	if loop {
		go am.handleMusicLoop(music)
	}

	return nil
}

func (am *AudioManager) handleMusicLoop(music *Music) {
	for {
		time.Sleep(100 * time.Millisecond)

		music.mutex.Lock()
		if music.player == nil || !music.loop {
			music.mutex.Unlock()
			break
		}

		if !music.player.IsPlaying() {

			music.player.Rewind()
			music.player.Play()
		}
		music.mutex.Unlock()
	}
}

func (am *AudioManager) StopMusic() {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	am.stopCurrentMusic()
}

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

func (am *AudioManager) SetMasterVolume(volume float64) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	am.masterVolume = clampVolume(volume)
	am.updateAllVolumes()
}

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

func (am *AudioManager) SetSoundVolume(volume float64) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	am.soundVolume = clampVolume(volume)
}

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

func (am *AudioManager) updateAllVolumes() {

	if am.currentMusic != nil {
		am.currentMusic.mutex.Lock()
		if am.currentMusic.player != nil {
			finalVolume := am.masterVolume * am.musicVolume * am.currentMusic.volume
			am.currentMusic.player.SetVolume(finalVolume)
		}
		am.currentMusic.mutex.Unlock()
	}

}

func (am *AudioManager) GetMasterVolume() float64 {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	return am.masterVolume
}

func (am *AudioManager) GetMusicVolume() float64 {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	return am.musicVolume
}

func (am *AudioManager) GetSoundVolume() float64 {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	return am.soundVolume
}

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

func (am *AudioManager) GetSoundNames() []string {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	names := make([]string, 0, len(am.sounds))
	for name := range am.sounds {
		names = append(names, name)
	}
	return names
}

func (am *AudioManager) GetSoundInfo(name string) (bool, int, error) {
	am.mutex.RLock()
	sound, exists := am.sounds[name]
	am.mutex.RUnlock()

	if !exists {
		return false, 0, fmt.Errorf("sound %s not found", name)
	}

	return true, len(sound.data), nil
}

func (am *AudioManager) CreateTestTone(name string, frequency float64, duration time.Duration) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	samples := int(float64(SampleRate) * duration.Seconds())
	data := make([]byte, samples*4)

	for i := 0; i < samples; i++ {

		t := float64(i) / float64(SampleRate)
		sample := int16(32767 * 0.1 * math.Sin(2*math.Pi*frequency*t))

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

func (am *AudioManager) Update() {

	am.mutex.RLock()
	defer am.mutex.RUnlock()

	for _, sound := range am.sounds {
		sound.mutex.Lock()
		am.cleanupSoundPlayers(sound)
		sound.mutex.Unlock()
	}
}

func (am *AudioManager) Cleanup() {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	am.stopCurrentMusic()

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

func clampVolume(volume float64) float64 {
	if volume < 0.0 {
		return 0.0
	}
	if volume > 1.0 {
		return 1.0
	}
	return volume
}

var globalAudioManager *AudioManager

func InitAudio(props *AudioProps) {
	globalAudioManager = NewAudioManager(props)
}

func GetAudioManager() *AudioManager {
	if globalAudioManager == nil {
		InitAudio(nil)
	}
	return globalAudioManager
}

func LoadSound(name string, fs embed.FS, filePath string) error {
	return GetAudioManager().LoadSoundFromFS(name, fs, filePath)
}

func LoadMusic(name string, fs embed.FS, filePath string) error {
	return GetAudioManager().LoadMusicFromFS(name, fs, filePath)
}

func PlaySound(name string) error {
	return GetAudioManager().PlaySound(name)
}

func PlaySoundWithVolume(name string, volume float64) error {
	return GetAudioManager().PlaySoundWithVolume(name, volume)
}

func PlayMusic(name string) error {
	return GetAudioManager().PlayMusic(name)
}

func PlayMusicWithOptions(name string, loop bool, volume float64) error {
	return GetAudioManager().PlayMusicWithOptions(name, loop, volume)
}

func StopMusic() {
	GetAudioManager().StopMusic()
}

func PauseMusic() {
	GetAudioManager().PauseMusic()
}

func ResumeMusic() {
	GetAudioManager().ResumeMusic()
}
