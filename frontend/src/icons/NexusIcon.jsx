import React from "react";
import nexusLogo from "@/assets/images/ic_nexus_logo.png";
export const NexusIcon = ({ width = 48, height = 48, ...props }) => (
  <img
    src={nexusLogo}
    width={width}
    height={height}
    alt="Bedrock Nexus logo"
    draggable="false"
    {...props}
  />
);
