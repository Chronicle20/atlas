// services/atlas-ui/src/lib/hooks/api/useCreateAndPollAccount.ts
import { useCallback, useEffect, useRef, useState } from "react";
import { accountsService } from "@/services/api/accounts.service";
import type { Tenant } from "@/types/models/tenant";
import { createErrorFromUnknown } from "@/types/api/errors";

export type CreateAndPollStatus =
  | "idle"
  | "submitting"
  | "polling"
  | "success"
  | "timeout"
  | "error";

export type CreateAndPollErrorKind = "duplicate-name" | "generic" | null;

export interface CreateAndPollResult {
  status: CreateAndPollStatus;
  accountId: number | null;
  errorMessage: string | null;
  errorKind: CreateAndPollErrorKind;
  submit(input: { name: string; password: string }): Promise<void>;
  retry(): Promise<void>;
  reset(): void;
}

const POLL_INTERVAL_MS = 1000;
const POLL_TIMEOUT_MS = 30_000;

export function useCreateAndPollAccount(tenant: Tenant): CreateAndPollResult {
  const [status, setStatus] = useState<CreateAndPollStatus>("idle");
  const [accountId, setAccountId] = useState<number | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [errorKind, setErrorKind] = useState<CreateAndPollErrorKind>(null);

  const submittedNameRef = useRef<string | null>(null);
  const submittedTenantIdRef = useRef<string | null>(null);
  const abortRef = useRef<AbortController | null>(null);
  const tenantIdRef = useRef<string | null>(tenant?.id ?? null);

  // Track the current tenant id so the polling loop can detect a swap.
  useEffect(() => {
    tenantIdRef.current = tenant?.id ?? null;
  }, [tenant?.id]);

  const reset = useCallback(() => {
    abortRef.current?.abort();
    abortRef.current = null;
    submittedNameRef.current = null;
    submittedTenantIdRef.current = null;
    setStatus("idle");
    setAccountId(null);
    setErrorMessage(null);
    setErrorKind(null);
  }, []);

  // If the active tenant changes mid-flight, abort and reset.
  useEffect(() => {
    if (
      submittedTenantIdRef.current &&
      tenant?.id &&
      tenant.id !== submittedTenantIdRef.current &&
      (status === "submitting" || status === "polling" || status === "timeout")
    ) {
      reset();
    }
  }, [tenant?.id, status, reset]);

  const poll = useCallback(
    async (name: string, controller: AbortController, capturedTenantId: string) => {
      setStatus("polling");
      const deadline = Date.now() + POLL_TIMEOUT_MS;
      while (Date.now() < deadline) {
        await new Promise<void>((res) => setTimeout(res, POLL_INTERVAL_MS));
        if (controller.signal.aborted) return;
        if (tenantIdRef.current !== capturedTenantId) return;
        try {
          const accounts = await accountsService.getAllAccounts({ name });
          if (controller.signal.aborted) return;
          if (tenantIdRef.current !== capturedTenantId) return;
          const found = accounts.find((a) => a.attributes.name === name);
          if (found) {
            setAccountId(Number(found.id));
            setStatus("success");
            return;
          }
        } catch {
          // transient — keep polling within the budget
        }
      }
      if (controller.signal.aborted) return;
      if (tenantIdRef.current !== capturedTenantId) return;
      setStatus("timeout");
    },
    []
  );

  const submit = useCallback(
    async ({ name, password }: { name: string; password: string }) => {
      abortRef.current?.abort();
      const controller = new AbortController();
      abortRef.current = controller;
      submittedNameRef.current = name;
      submittedTenantIdRef.current = tenant?.id ?? null;
      setAccountId(null);
      setErrorMessage(null);
      setErrorKind(null);
      setStatus("submitting");

      try {
        await accountsService.createAccount(tenant, { name, password });
      } catch (err) {
        if (controller.signal.aborted) return;
        const status = (err as { status?: number })?.status;
        const message = createErrorFromUnknown(err).message;
        setErrorMessage(message);
        setErrorKind(status === 409 ? "duplicate-name" : "generic");
        setStatus("error");
        return;
      }
      if (controller.signal.aborted) return;
      await poll(name, controller, submittedTenantIdRef.current ?? "");
    },
    [tenant, poll]
  );

  const retry = useCallback(async () => {
    if (!submittedNameRef.current) return;
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    setErrorMessage(null);
    setErrorKind(null);
    await poll(submittedNameRef.current, controller, submittedTenantIdRef.current ?? "");
  }, [poll]);

  // Abort any in-flight work on unmount.
  useEffect(() => {
    return () => {
      abortRef.current?.abort();
    };
  }, []);

  return { status, accountId, errorMessage, errorKind, submit, retry, reset };
}
