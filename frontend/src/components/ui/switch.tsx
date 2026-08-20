import { Switch as SwitchPrimitive } from "@base-ui/react/switch"

import { cn } from "@/lib/utils"

function Switch({ className, ...props }: SwitchPrimitive.Root.Props) {
  return (
    <SwitchPrimitive.Root
      data-slot="switch"
      className={cn(
        "relative inline-flex h-6 w-11 shrink-0 items-center rounded-full bg-tinta/20 transition-colors data-[checked]:bg-acento disabled:cursor-not-allowed disabled:opacity-50",
        className
      )}
      {...props}
    >
      <SwitchPrimitive.Thumb
        data-slot="switch-thumb"
        className="block size-5 translate-x-0.5 rounded-full bg-superficie shadow transition-transform data-[checked]:translate-x-[22px]"
      />
    </SwitchPrimitive.Root>
  )
}

export { Switch }
