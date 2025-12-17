<template>
  <button
    type="button"
    :class="toggleClasses"
    :disabled="disabled"
    @click="handleToggle"
    role="switch"
    :aria-checked="modelValue"
  >
    <span :class="switchClasses" />
  </button>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  disabled: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:modelValue', 'change'])

const handleToggle = () => {
  if (!props.disabled) {
    emit('update:modelValue', !props.modelValue)
    emit('change', !props.modelValue)
  }
}

const toggleClasses = computed(() => {
  const baseClasses = 'relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2'
  const activeClasses = props.modelValue ? 'bg-green-600 focus:ring-green-500' : 'bg-gray-300 focus:ring-gray-500'
  const disabledClasses = props.disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'

  return `${baseClasses} ${activeClasses} ${disabledClasses}`
})

const switchClasses = computed(() => {
  const baseClasses = 'inline-block h-4 w-4 transform rounded-full bg-white transition-transform'
  const positionClasses = props.modelValue ? 'translate-x-6' : 'translate-x-1'

  return `${baseClasses} ${positionClasses}`
})
</script>
