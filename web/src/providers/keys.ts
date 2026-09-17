import type { InjectionKey } from 'vue'
import type { AuthProvider } from './auth'
import type { SelectionProvider } from './selection'
import type { FeedsTreeProvider } from './feedsTree'
import type { ItemsProvider } from './items'

export const authKey: InjectionKey<AuthProvider> = Symbol('auth')
export const selectionKey: InjectionKey<SelectionProvider> = Symbol('selection')
export const feedsTreeKey: InjectionKey<FeedsTreeProvider> = Symbol('feedsTree')
export const itemsKey: InjectionKey<ItemsProvider> = Symbol('items')
