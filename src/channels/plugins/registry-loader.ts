import type { PluginChannelRegistration, PluginRegistry } from "../../plugins/registry.js";
import { getActivePluginRegistry, getActivePluginRegistryVersion } from "../../plugins/runtime.js";
import type { ChannelId } from "./types.js";

type ChannelRegistryValueResolver<TValue> = (
  entry: PluginChannelRegistration,
) => TValue | undefined;

export function createChannelRegistryLoader<TValue>(
  resolveValue: ChannelRegistryValueResolver<TValue>,
): (id: ChannelId) => Promise<TValue | undefined> {
  const cache = new Map<ChannelId, TValue>();
  let lastRegistry: PluginRegistry | null = null;
  let lastRegistryVersion: number = -1;

  return async (id: ChannelId): Promise<TValue | undefined> => {
    const registry = getActivePluginRegistry();
    const registryVersion = getActivePluginRegistryVersion();
    if (registry !== lastRegistry || registryVersion !== lastRegistryVersion) {
      cache.clear();
      lastRegistry = registry;
      lastRegistryVersion = registryVersion;
    }
    const cached = cache.get(id);
    if (cached) {
      return cached;
    }
    const pluginEntry = registry?.channels.find((entry) => entry.plugin.id === id);
    if (!pluginEntry) {
      cache.delete(id);
      return undefined;
    }
    const resolved = resolveValue(pluginEntry);
    if (resolved) {
      cache.set(id, resolved);
    }
    return resolved;
  };
}
