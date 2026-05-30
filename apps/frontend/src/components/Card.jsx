// White floating card used across the dark canvas.
export default function Card({ className = '', children, ...rest }) {
  return (
    <div
      className={`rounded-2xl bg-white shadow-sm ${className}`}
      {...rest}
    >
      {children}
    </div>
  )
}
