import { ref } from 'vue'

type Fn = (...args: any[]) => Promise<any>

export type UseLoadingResult<T> = T & {
  isLoading: boolean
}

export function useLoading<T extends Fn>(fn: T): UseLoadingResult<T> {
  const executingCount = ref(0)

  const wrapperFn = async (...args: Parameters<T>) => {
    try {
      executingCount.value++

      const result = await fn(...args)
      return result
    } finally {
      executingCount.value--
    }
  }

  Object.defineProperty(wrapperFn, 'isLoading', {
    get() {
      return executingCount.value > 0
    },
  })

  return wrapperFn as UseLoadingResult<T>
}
