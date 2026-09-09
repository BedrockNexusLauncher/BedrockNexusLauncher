import React from "react";
import { useTranslation } from "react-i18next";
import {
  Button,
  Chip,
  Image,
  Input,
  Modal,
  ModalBody,
  ModalContent,
  ModalFooter,
  ModalHeader,
  Progress,
  Select,
  SelectItem,
} from "@heroui/react";
import { FaPlus, FaSave, FaTrash, FaUpload } from "react-icons/fa";

export type SkinPackDraft = {
  name: string;
  geometry: string;
  data: string;
};

type Props = {
  isOpen: boolean;
  mode: "create" | "edit";
  initialName?: string;
  initialSkins?: SkinPackDraft[];
  isSaving?: boolean;
  onOpenChange: (open: boolean) => void;
  onSave: (name: string, skins: SkinPackDraft[]) => void;
};

const fileToBase64 = (file: File): Promise<string> =>
  new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result || "").split(",")[1] || "");
    reader.onerror = () => reject(reader.error || new Error("Unable to read image"));
    reader.readAsDataURL(file);
  });

const skinGeometries = [
  { key: "geometry.humanoid.custom", labelKey: "skinpack.classic" },
  { key: "geometry.humanoid.customSlim", labelKey: "skinpack.slim" },
] as const;

export function SkinPackBuilderModal({
  isOpen,
  mode,
  initialName = "Custom Skin Pack",
  initialSkins = [],
  isSaving = false,
  onOpenChange,
  onSave,
}: Props) {
  const { t } = useTranslation();
  const [name, setName] = React.useState(initialName);
  const [skins, setSkins] = React.useState<SkinPackDraft[]>(initialSkins);
  const [busy, setBusy] = React.useState(false);
  const [replaceIndex, setReplaceIndex] = React.useState<number | null>(null);
  const inputRef = React.useRef<HTMLInputElement | null>(null);

  React.useEffect(() => {
    if (isOpen) {
      setName(initialName);
      setSkins(initialSkins);
      setBusy(false);
      setReplaceIndex(null);
    }
  }, [initialName, initialSkins, isOpen]);

  const acceptFiles = async (files: FileList | null) => {
    if (!files?.length) return;
    setBusy(true);
    try {
      const next = [...skins];
      for (const file of Array.from(files)) {
        if (!file.type.startsWith("image/png")) continue;
        const data = await fileToBase64(file);
        const entry = {
          name: file.name.replace(/\.png$/i, "") || `Skin ${next.length + 1}`,
          geometry:
            replaceIndex !== null
              ? next[replaceIndex].geometry
              : "geometry.humanoid.custom",
          data,
        };
        if (replaceIndex !== null) {
          next[replaceIndex] = entry;
          setReplaceIndex(null);
          break;
        }
        next.push(entry);
      }
      setSkins(next);
    } finally {
      setBusy(false);
      if (inputRef.current) inputRef.current.value = "";
    }
  };

  const updateSkin = (index: number, patch: Partial<SkinPackDraft>) => {
    setSkins((current) =>
      current.map((skin, itemIndex) =>
        itemIndex === index ? { ...skin, ...patch } : skin,
      ),
    );
  };

  return (
    <Modal
      isOpen={isOpen}
      onOpenChange={onOpenChange}
      size="3xl"
      scrollBehavior="inside"
      backdrop="blur"
    >
      <ModalContent>
        {(onClose) => (
          <>
            <ModalHeader className="flex flex-col gap-1">
              <div className="flex items-center gap-2">
                {mode === "create" ? t("skinpack.create_title") : t("skinpack.edit_title")}
                <Chip size="sm" color="warning" variant="flat">
                  {t("skinpack.beta")}
                </Chip>
              </div>
              <span className="text-sm font-normal text-default-500">
                {t("skinpack.description")}
              </span>
            </ModalHeader>
            <ModalBody>
              <div className="flex flex-col gap-4">
                <Input
                  label={t("skinpack.pack_name")}
                  value={name}
                  onValueChange={setName}
                  isDisabled={isSaving}
                  variant="bordered"
                />
                <input
                  ref={inputRef}
                  type="file"
                  accept="image/png"
                  multiple={replaceIndex === null}
                  className="hidden"
                  onChange={(event) => void acceptFiles(event.target.files)}
                />
                <Button
                  color="primary"
                  variant="flat"
                  startContent={replaceIndex === null ? <FaPlus /> : <FaUpload />}
                  isDisabled={busy || isSaving}
                  onPress={() => inputRef.current?.click()}
                >
                  {replaceIndex === null ? t("skinpack.add_skins") : t("skinpack.choose_replacement")}
                </Button>
                {busy && <Progress size="sm" isIndeterminate aria-label="Reading skins" />}
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  {skins.map((skin, index) => (
                    <div
                      key={`${index}-${skin.name}`}
                      className="flex gap-3 rounded-xl border border-default-200 dark:border-zinc-700 p-3"
                    >
                      <Image
                        src={`data:image/png;base64,${skin.data}`}
                        alt={skin.name}
                        className="w-20 h-20 object-contain rounded-lg bg-default-100 shrink-0"
                        radius="none"
                      />
                      <div className="min-w-0 flex-1 flex flex-col gap-2">
                        <Input
                          size="sm"
                          label={t("skinpack.skin_name")}
                          value={skin.name}
                          onValueChange={(value) => updateSkin(index, { name: value })}
                          isDisabled={isSaving}
                        />
                        <Select
                          size="sm"
                          label={t("skinpack.model")}
                          selectedKeys={new Set([skin.geometry])}
                          onSelectionChange={(keys) => {
                            const selectedGeometry = String(Array.from(keys)[0] || "geometry.humanoid.custom");
                            const geometry = selectedGeometry.includes("Slim")
                              ? "geometry.humanoid.customSlim"
                              : selectedGeometry;
                            updateSkin(index, { geometry });
                          }}
                          isDisabled={isSaving}
                        >
                          {skinGeometries.map((option) => (
                            <SelectItem key={option.key}>
                              {t(option.labelKey)}
                            </SelectItem>
                          ))}
                        </Select>
                        <div className="flex gap-2">
                          <Button
                            size="sm"
                            variant="flat"
                            startContent={<FaUpload />}
                            onPress={() => {
                              setReplaceIndex(index);
                              inputRef.current?.click();
                            }}
                            isDisabled={isSaving}
                          >
                            {t("skinpack.replace")}
                          </Button>
                          <Button
                            isIconOnly
                            size="sm"
                            color="danger"
                            variant="flat"
                            aria-label={t("skinpack.remove")}
                            onPress={() => setSkins((current) => current.filter((_, itemIndex) => itemIndex !== index))}
                            isDisabled={isSaving}
                          >
                            <FaTrash />
                          </Button>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
                {!skins.length && (
                  <div className="rounded-xl border border-dashed border-default-300 p-8 text-center text-default-500">
                    {t("skinpack.no_skins")}
                  </div>
                )}
              </div>
            </ModalBody>
            <ModalFooter>
              <Button variant="light" onPress={onClose} isDisabled={isSaving}>
                {t("common.cancel")}
              </Button>
              <Button
                color="primary"
                startContent={<FaSave />}
                isLoading={isSaving}
                isDisabled={!name.trim() || skins.length === 0 || busy}
                onPress={() => onSave(name.trim(), skins)}
              >
                {mode === "create" ? t("skinpack.build_install") : t("skinpack.save_install")}
              </Button>
            </ModalFooter>
          </>
        )}
      </ModalContent>
    </Modal>
  );
}
