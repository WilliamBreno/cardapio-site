import { useRef, useState, type ChangeEvent, type DragEvent } from "react"
import { Camera } from "lucide-react"

import { cn } from "@/lib/utils"

interface DropzoneProps {
  onFileChange: (e: ChangeEvent<HTMLInputElement>) => void
  previewUrl?: string
  className?: string
  disabled?: boolean
  texto?: string
}

function Dropzone({ onFileChange, previewUrl, className, disabled, texto = "Arraste a foto ou clique pra escolher" }: DropzoneProps) {
  const [sobre, setSobre] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  function onDrop(e: DragEvent<HTMLDivElement>) {
    e.preventDefault()
    setSobre(false)
    const arquivo = e.dataTransfer.files?.[0]
    if (!arquivo || !inputRef.current) return
    // Reconstrói um FileList no input escondido e dispara um evento
    // "change" nativo nele, pra reaproveitar o MESMO handler que já
    // processa o upload via seleção manual de arquivo — evita duplicar a
    // lógica de envio pra Cloudinary em dois lugares.
    const dt = new DataTransfer()
    dt.items.add(arquivo)
    inputRef.current.files = dt.files
    inputRef.current.dispatchEvent(new Event("change", { bubbles: true }))
  }

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={() => !disabled && inputRef.current?.click()}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") inputRef.current?.click()
      }}
      onDragOver={(e) => {
        e.preventDefault()
        setSobre(true)
      }}
      onDragLeave={() => setSobre(false)}
      onDrop={onDrop}
      data-slot="dropzone"
      className={cn(
        "flex cursor-pointer flex-col items-center gap-2 rounded-xl border-2 border-dashed border-tinta/20 bg-fundo px-6 py-8 text-center transition",
        sobre && "border-acento bg-acento/5",
        disabled && "cursor-not-allowed opacity-60",
        className
      )}
    >
      {previewUrl ? (
        <img src={previewUrl} alt="Prévia" className="size-16 rounded-full object-cover" />
      ) : (
        <Camera className="size-6 text-tinta-suave" />
      )}
      <p className="text-sm text-tinta-suave">{texto}</p>
      <input ref={inputRef} type="file" accept="image/*" onChange={onFileChange} disabled={disabled} className="hidden" />
    </div>
  )
}

export { Dropzone }
