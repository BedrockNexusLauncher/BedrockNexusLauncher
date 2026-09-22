import React from "react";
import {
  Button,
  Input,
  Progress,
  Select,
  SelectItem,
  Tabs,
  Tab,
  Chip,
} from "@heroui/react";
import { useTranslation } from "react-i18next";
import { UnifiedModal } from "@/components/UnifiedModal";
import { COMPONENT_STYLES } from "@/constants/componentStyles";
import { cn } from "@/utils/cn";
import { useUwpInstaller } from "@/hooks/useUwpInstaller";

interface UwpInstallModalProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  onInstalled?: () => void;
}

export const UwpInstallModal: React.FC<UwpInstallModalProps> = ({
  isOpen,
  onOpenChange,
  onInstalled,
}) => {
  const { t } = useTranslation();
  const [channel, setChannel] = React.useState("release");
  const [updateID, setUpdateID] = React.useState("");
  const [instanceName, setInstanceName] = React.useState("");
  const installer = useUwpInstaller(onInstalled);

  React.useEffect(() => {
    if (isOpen) {
      installer.reset();
      setUpdateID("");
      setInstanceName("");
      installer.loadChannel(channel);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen]);

  React.useEffect(() => {
    if (isOpen) {
      setUpdateID("");
      installer.loadChannel(channel);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [channel]);

  const busy =
    installer.step === "downloading" || installer.step === "installing";
  const pct =
    installer.progress && installer.progress.total > 0
      ? Math.min(
          100,
          (installer.progress.downloaded / installer.progress.total) * 100,
        )
      : 0;

  const close = () => {
    if (busy) return;
    onOpenChange(false);
  };

  return (
    <UnifiedModal
      isOpen={isOpen}
      onOpenChange={(o) => {
        if (!o) close();
      }}
      title="Install UWP instance"
      size="lg"
      footer={
        <div className="flex w-full justify-end gap-2">
          {busy ? (
            <Button color="danger" variant="flat" onPress={() => installer.cancel()}>
              {t("common.cancel")}
            </Button>
          ) : (
            <>
              <Button variant="flat" onPress={close}>
                {t("common.close")}
              </Button>
              <Button
                color="primary"
                isDisabled={!updateID || !instanceName.trim()}
                onPress={() => installer.startInstall(updateID, instanceName)}
              >
                Install
              </Button>
            </>
          )}
        </div>
      }
    >
      <div className="flex flex-col gap-4">
        {installer.devMode === false && (
          <Chip color="warning" variant="flat" className="self-start">
            Windows Developer Mode is off — enable it to install UWP versions.
          </Chip>
        )}
        <Tabs
          aria-label="UWP channel"
          selectedKey={channel}
          onSelectionChange={(k) => setChannel(String(k))}
          variant="solid"
          classNames={COMPONENT_STYLES.tabs}
          isDisabled={busy}
        >
          <Tab key="release" title="Release" />
          <Tab key="beta" title="Beta" />
          <Tab key="preview" title="Preview" />
        </Tabs>
        <Select
          label="Version"
          placeholder="Select a version"
          selectedKeys={updateID ? new Set([updateID]) : new Set()}
          onSelectionChange={(keys) => {
            const v = Array.from(keys)[0] as string;
            setUpdateID(v || "");
          }}
          isDisabled={busy || installer.entries.length === 0}
          classNames={COMPONENT_STYLES.input}
        >
          {installer.entries.map((e) => (
            <SelectItem key={e.uuid} textValue={e.version}>
              {e.version}
            </SelectItem>
          ))}
        </Select>
        <Input
          label="Instance name"
          placeholder="my-uwp"
          value={instanceName}
          onValueChange={setInstanceName}
          isDisabled={busy}
          classNames={COMPONENT_STYLES.input}
        />
        {installer.step === "downloading" && (
          <div className="flex flex-col gap-2">
            <Progress
              aria-label="UWP download progress"
              value={pct}
              showValueLabel
              className={cn("w-full")}
            />
          </div>
        )}
        {installer.step === "installing" && (
          <Chip color="primary" variant="flat" className="self-start">
            Registering package…
          </Chip>
        )}
        {installer.step === "success" && (
          <Chip color="success" variant="flat" className="self-start">
            Installed — it now appears in your instances.
          </Chip>
        )}
        {installer.step === "error" && installer.installError && (
          <Chip color="danger" variant="flat" className="self-start">
            {installer.installError}
          </Chip>
        )}
      </div>
    </UnifiedModal>
  );
};
