import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button, Card, CardBody } from "@heroui/react";
import { SectionHeader } from "@/components/PageHeader";
import {
  FaGithub,
  FaCode,
  FaStar,
  FaBug,
  FaRocket,
  FaUserAstronaut,
} from "react-icons/fa";
import { Browser } from "@wailsio/runtime";
import { motion, Variants } from "framer-motion";
import { PageContainer } from "@/components/PageContainer";
import { ReportProblemModal } from "@/components/ReportProblemModal";
import { LAYOUT } from "@/constants/layout";
import { cn } from "@/utils/cn";
import creatorAvatar from "@/assets/images/Avatar.png";

export default function AboutPage() {
  const { t } = useTranslation();
  const [isAnimating, setIsAnimating] = useState(true);
  const [reportOpen, setReportOpen] = useState(false);

  const repoUrl = "https://github.com/BedrockNexusLauncher/BedrockNexusLauncher";

  const cardVariants: Variants = {
    hidden: { opacity: 0, y: 20 },
    visible: (i: number) => ({
      opacity: 1,
      y: 0,
      transition: {
        delay: i * 0.1,
        duration: 0.4,
        ease: "easeOut",
      },
    }),
  };

  return (
    <PageContainer
      className={cn("relative", isAnimating && "overflow-hidden")}
      animate={false}
    >
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Launcher Creator Section */}
        <motion.div
          custom={0}
          initial="hidden"
          animate="visible"
          variants={cardVariants}
          className="lg:col-span-2"
        >
          <Card className={cn("h-full", LAYOUT.GLASS_CARD.BASE)}>
            <CardBody className="p-6">
              <SectionHeader
                className="mb-4"
                icon={<FaUserAstronaut size={20} />}
                iconWrapperClassName="bg-primary-500/10 text-primary-600 dark:text-primary-400"
                title={t("about.authors")}
              />
              <div className="flex items-center gap-4 rounded-2xl border border-default-100 dark:border-white/10 bg-default-50 dark:bg-white/5 p-4">
                <img
                  src={creatorAvatar}
                  alt={t("about.creator_name")}
                  className="w-16 h-16 shrink-0 rounded-full object-cover ring-2 ring-primary-500/30"
                />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-lg font-bold text-default-800 dark:text-zinc-100">
                      {t("about.creator_name")}
                    </span>
                    <span className="text-[11px] px-2 py-0.5 rounded-full bg-primary-500/15 text-primary-600 dark:text-primary-400 font-medium">
                      {t("about.author")}
                    </span>
                  </div>
                  <p className="rtl-paragraph text-small text-default-500 dark:text-zinc-400 leading-relaxed">
                    {t("about.creator_desc")}
                  </p>
                </div>
              </div>
            </CardBody>
          </Card>
        </motion.div>

        {/* About the Launcher Section */}
        <motion.div
          custom={1}
          initial="hidden"
          animate="visible"
          variants={cardVariants}
          className="lg:col-span-2"
        >
          <Card className={cn("h-full", LAYOUT.GLASS_CARD.BASE)}>
            <CardBody className="p-6">
              <SectionHeader
                className="mb-4"
                icon={<FaRocket size={20} />}
                iconWrapperClassName="bg-primary-500/10 text-primary-600 dark:text-primary-400"
                title={t("about.launcher")}
              />
              <p className="rtl-paragraph text-default-600 dark:text-zinc-400 leading-relaxed">
                {t("about.launcher.desc")}
              </p>
            </CardBody>
          </Card>
        </motion.div>

        {/* Source Code Section */}
        <motion.div
          custom={2}
          initial="hidden"
          animate="visible"
          variants={cardVariants}
          className="lg:col-span-2"
          onAnimationComplete={() => setIsAnimating(false)}
        >
          <Card className={cn("h-full", LAYOUT.GLASS_CARD.BASE)}>
            <CardBody className="p-6">
              <div className="flex flex-col md:flex-row md:items-start justify-between gap-4">
                <div className="flex-1">
                  <SectionHeader
                    className="mb-4"
                    icon={<FaCode size={20} />}
                    iconWrapperClassName="bg-blue-500/10 text-blue-600 dark:text-blue-400"
                    title={t("about.source")}
                  />
                  <div className="flex flex-wrap items-center gap-3 mb-4">
                    <Button
                      variant="flat"
                      className="bg-default-100 dark:bg-white/10"
                      startContent={<FaGithub className="text-lg" />}
                      onPress={() => Browser.OpenURL(repoUrl)}
                    >
                      {t("about.github_repo")}
                    </Button>
                  </div>
                  <p className="rtl-paragraph text-small text-default-500 dark:text-zinc-400">
                    {t("about.license.tip")}
                  </p>
                </div>

                <div className="flex-1 border-t md:border-t-0 md:border-s border-default-100 dark:border-white/5 pt-6 md:pt-0 md:ps-6">
                  <SectionHeader
                    className="mb-4"
                    icon={<FaGithub size={20} />}
                    iconWrapperClassName="bg-default-500/10 text-default-600 dark:text-zinc-400"
                    title={t("about.contribute")}
                  />
                  <p className="rtl-paragraph text-default-600 dark:text-zinc-400 leading-relaxed mb-4">
                    {t("about.contribute.desc")}
                  </p>
                  <div className="flex gap-2">
                    <Button
                      size="sm"
                      variant="flat"
                      className="bg-default-100 dark:bg-white/10"
                      startContent={<FaBug />}
                      onPress={() => setReportOpen(true)}
                    >
                      {t("reportProblem.button")}
                    </Button>
                    <Button
                      size="sm"
                      variant="flat"
                      className="bg-default-100 dark:bg-white/10"
                      startContent={<FaGithub />}
                      onPress={() => Browser.OpenURL(`${repoUrl}/issues`)}
                    >
                      {t("about.issue")}
                    </Button>
                    <Button
                      size="sm"
                      color="primary"
                      className="bg-linear-to-r from-primary-500 to-primary-400 text-white shadow-lg shadow-primary-900/20"
                      startContent={<FaStar />}
                      onPress={() => Browser.OpenURL(`${repoUrl}`)}
                    >
                      {t("about.star_fork")}
                    </Button>
                  </div>
                </div>
              </div>
            </CardBody>
          </Card>
        </motion.div>
      </div>

      <ReportProblemModal isOpen={reportOpen} onOpenChange={setReportOpen} />
    </PageContainer>
  );
}
