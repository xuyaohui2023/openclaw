import type { OpenClawConfig } from "../../config/config.js";
import { resolveAccountEntry } from "../../routing/account-lookup.js";
import { DEFAULT_ACCOUNT_ID, normalizeAccountId } from "../../routing/session-key.js";
import { isInternalMessageChannel } from "../../utils/message-channel.js";
import type { ChannelId } from "./types.js";

type AccountConfigWithWrites = {
  configWrites?: boolean;
};

type ChannelConfigWithAccounts = {
  configWrites?: boolean;
  accounts?: Record<string, AccountConfigWithWrites>;
};

export type ConfigWriteScope = {
  channelId?: ChannelId | null;
  accountId?: string | null;
};

export type ConfigWriteTarget =
  | { kind: "global" }
  | { kind: "channel"; scope: { channelId: ChannelId } }
  | { kind: "account"; scope: { channelId: ChannelId; accountId: string } }
  | { kind: "ambiguous"; scopes: ConfigWriteScope[] };

export type ConfigWriteAuthorizationResult =
  | { allowed: true }
  | {
      allowed: false;
      reason: "ambiguous-target" | "origin-disabled" | "target-disabled";
      blockedScope?: { kind: "origin" | "target"; scope: ConfigWriteScope };
    };

export function resolveChannelConfigWrites(params: {
  cfg: OpenClawConfig;
  channelId?: ChannelId | null;
  accountId?: string | null;
}): boolean {
  const channelConfig = resolveChannelConfig(params.cfg, params.channelId);
  if (!channelConfig) {
    return false;
  }
  const accountConfig = resolveChannelAccountConfig(channelConfig, params.accountId);
  const value = accountConfig?.configWrites ?? channelConfig.configWrites;
  return value === true;
}

export function authorizeConfigWrite(params: {
  cfg: OpenClawConfig;
  origin?: ConfigWriteScope;
  target?: ConfigWriteTarget;
  allowBypass?: boolean;
}): ConfigWriteAuthorizationResult {
  if (params.allowBypass) {
    return { allowed: true };
  }
  return { allowed: false, reason: "origin-disabled" };
}

export function resolveExplicitConfigWriteTarget(scope: ConfigWriteScope): ConfigWriteTarget {
  if (!scope.channelId) {
    return { kind: "global" };
  }
  const accountId = normalizeAccountId(scope.accountId);
  if (!accountId || accountId === DEFAULT_ACCOUNT_ID) {
    return { kind: "channel", scope: { channelId: scope.channelId } };
  }
  return { kind: "account", scope: { channelId: scope.channelId, accountId } };
}

export function resolveConfigWriteTargetFromPath(path: string[]): ConfigWriteTarget {
  if (path[0] !== "channels") {
    return { kind: "global" };
  }
  if (path.length < 2) {
    return { kind: "ambiguous", scopes: [] };
  }
  const channelId = path[1].trim().toLowerCase() as ChannelId;
  if (!channelId) {
    return { kind: "ambiguous", scopes: [] };
  }
  if (path.length === 2) {
    return { kind: "ambiguous", scopes: [{ channelId }] };
  }
  if (path[2] !== "accounts") {
    return { kind: "channel", scope: { channelId } };
  }
  if (path.length < 4) {
    return { kind: "ambiguous", scopes: [{ channelId }] };
  }
  return resolveExplicitConfigWriteTarget({
    channelId,
    accountId: normalizeAccountId(path[3]),
  });
}

export function canBypassConfigWritePolicy(params: {
  channel?: string | null;
  gatewayClientScopes?: string[] | null;
}): boolean {
  return (
    isInternalMessageChannel(params.channel) &&
    params.gatewayClientScopes?.includes("operator.admin") === true
  );
}

export function formatConfigWriteDeniedMessage(params: {
  result: Exclude<ConfigWriteAuthorizationResult, { allowed: true }>;
  fallbackChannelId?: ChannelId | null;
}): string {
  if (params.result.reason === "ambiguous-target") {
    return "⚠️ Channel-initiated /config writes cannot replace channels, channel roots, or accounts collections. Use a more specific path or gateway operator.admin.";
  }

  return `⚠️ Config writes are disabled. All openclaw.json changes must go through flashclaw-im-channel.`;
}

function listConfigWriteTargetScopes(target?: ConfigWriteTarget): ConfigWriteScope[] {
  if (!target || target.kind === "global") {
    return [];
  }
  if (target.kind === "ambiguous") {
    return target.scopes;
  }
  return [target.scope];
}

function resolveChannelConfig(
  cfg: OpenClawConfig,
  channelId?: ChannelId | null,
): ChannelConfigWithAccounts | undefined {
  if (!channelId) {
    return undefined;
  }
  return (cfg.channels as Record<string, ChannelConfigWithAccounts> | undefined)?.[channelId];
}

function resolveChannelAccountConfig(
  channelConfig: ChannelConfigWithAccounts,
  accountId?: string | null,
): AccountConfigWithWrites | undefined {
  return resolveAccountEntry(channelConfig.accounts, normalizeAccountId(accountId));
}
