import { useEffect, useRef } from 'react'

// A lightweight emoji picker (no external dependency). Curated common emojis
// grouped by category. Calls onSelect(emoji) and onClose when clicking outside.
const GROUPS = [
  {
    name: 'Smileys',
    emojis: '😀 😃 😄 😁 😆 😅 😂 🤣 😊 😇 🙂 🙃 😉 😌 😍 🥰 😘 😗 😋 😛 😝 😜 🤪 🤨 🧐 🤓 😎 🤩 🥳 😏 😒 😞 😔 😟 😕 🙁 😣 😖 😫 😩 🥺 😢 😭 😤 😠 😡 🤬 🤯 😳 🥵 🥶 😱 😨 😰 😥 😓 🤗 🤔 🤭 🤫 😶 😐 😑 😬 🙄 😴 🤤 😪 😵 🤐 🥴 🤢 🤮 🤧 😷 🤒 🤕',
  },
  {
    name: 'Gestures',
    emojis: '👍 👎 👌 ✌️ 🤞 🤟 🤘 👈 👉 👆 👇 ☝️ ✋ 🤚 🖐️ 🖖 👋 🤙 💪 🙏 🤝 👏 🙌 👐 🤲 🤜 🤛 ✊ 👊 ❤️ 🧡 💛 💚 💙 💜 🖤 🤍 🤎 💕 💞 💓 💗 💖 💘 💝',
  },
  {
    name: 'Objects',
    emojis: '🎉 🎊 🎈 🎁 🏆 🥇 ⭐ 🌟 ✨ ⚡ 🔥 💥 💯 ✅ ❌ ❓ ❗ 💡 📌 📎 🔔 📣 📢 💬 💭 🕐 ⏰ 📅 📆 💰 💳 🛒 📦 ✈️ 🚗 🏠 ☎️ 📱 💻 ⌨️ 🖥️ 📷 🎥 🎵 🎶',
  },
  {
    name: 'Nature',
    emojis: '🐶 🐱 🐭 🐹 🐰 🦊 🐻 🐼 🐨 🐯 🦁 🐮 🐷 🐸 🐵 🐔 🐧 🐦 🦄 🐝 🌸 🌹 🌻 🌷 🌼 🌳 🌲 🌴 🍀 🍁 🌍 🌙 ☀️ ⛅ ☁️ 🌧️ ⛄ 🌈 💧 🌊',
  },
  {
    name: 'Food',
    emojis: '🍎 🍊 🍋 🍌 🍉 🍇 🍓 🍒 🍑 🥭 🍍 🥥 🍅 🥑 🌽 🥕 🍔 🍟 🍕 🌭 🥪 🌮 🌯 🍜 🍝 🍣 🍱 🍚 🍛 🍦 🍰 🎂 🍫 🍬 🍭 🍩 🍪 ☕ 🍵 🍺 🍻 🥤',
  },
]

export default function EmojiPicker({ onSelect, onClose }) {
  const ref = useRef(null)

  useEffect(() => {
    const handler = (e) => {
      if (ref.current && !ref.current.contains(e.target)) onClose?.()
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [onClose])

  return (
    <div
      ref={ref}
      className="absolute bottom-14 left-0 z-20 w-72 rounded-2xl border border-gray-200 bg-white p-3 shadow-xl sm:w-80"
    >
      <div className="max-h-64 space-y-3 overflow-y-auto pr-1">
        {GROUPS.map((g) => (
          <div key={g.name}>
            <p className="mb-1 px-1 text-[11px] font-semibold uppercase tracking-wide text-gray-400">
              {g.name}
            </p>
            <div className="grid grid-cols-8 gap-0.5">
              {g.emojis.split(' ').map((e, i) => (
                <button
                  key={`${g.name}-${i}`}
                  type="button"
                  onClick={() => onSelect(e)}
                  className="flex h-8 w-8 items-center justify-center rounded-lg text-xl hover:bg-gray-100"
                >
                  {e}
                </button>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
