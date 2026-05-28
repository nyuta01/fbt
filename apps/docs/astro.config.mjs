import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";

const SITE = process.env.SITE ?? "https://nyuta01.github.io";
const BASE = process.env.BASE ?? "/fbt";

export default defineConfig({
  site: SITE,
  base: BASE,
  integrations: [
    starlight({
      title: "fbt",
      tagline: "Local-first file build tool for knowledge artifacts.",
      logo: {
        light: "./src/assets/logo-light.svg",
        dark: "./src/assets/logo-dark.svg",
        replacesTitle: false,
      },
      favicon: "/favicon.svg",
      head: [
        { tag: "meta", attrs: { property: "og:type", content: "website" } },
        {
          tag: "meta",
          attrs: { property: "og:image", content: SITE + BASE + "/og-image.svg" },
        },
        { tag: "meta", attrs: { property: "og:image:width", content: "1200" } },
        { tag: "meta", attrs: { property: "og:image:height", content: "630" } },
        { tag: "meta", attrs: { name: "twitter:card", content: "summary_large_image" } },
        {
          tag: "meta",
          attrs: { name: "twitter:image", content: SITE + BASE + "/og-image.svg" },
        },
      ],
      social: [
        { icon: "github", label: "GitHub", href: "https://github.com/nyuta01/fbt" },
      ],
      editLink: {
        baseUrl: "https://github.com/nyuta01/fbt/edit/main/apps/docs/",
      },
      tableOfContents: { minHeadingLevel: 2, maxHeadingLevel: 4 },
      customCss: [
        "./src/styles/tokens.css",
        "./src/styles/starlight-overrides.css",
      ],
      sidebar: [
        {
          label: "Introduction",
          items: [
            { label: "What is fbt", slug: "introduction/what-is-fbt" },
            { label: "Concepts", slug: "introduction/concepts" },
            { label: "Architecture", slug: "introduction/architecture" },
          ],
        },
        {
          label: "Get started",
          items: [
            { label: "Installation", slug: "get-started/installation" },
            { label: "Quickstart", slug: "get-started/quickstart" },
            { label: "Manual generation", slug: "get-started/manual-generation" },
          ],
        },
        {
          label: "CLI",
          items: [
            { label: "Overview", slug: "cli/overview" },
            { label: "Project loop", slug: "cli/project-loop" },
            { label: "Inspection", slug: "cli/inspection" },
          ],
        },
        {
          label: "Runners",
          items: [
            { label: "External runners", slug: "runners/external-runners" },
            { label: "OpenAI runner", slug: "runners/openai-runner" },
            { label: "Authoring contract", slug: "runners/authoring-contract" },
          ],
        },
        {
          label: "Lineage and standards",
          items: [
            { label: "Lineage model", slug: "standards/lineage-model" },
            { label: "OpenLineage export", slug: "standards/openlineage" },
            { label: "OpenTelemetry export", slug: "standards/opentelemetry" },
            { label: "Visualization", slug: "standards/visualization" },
          ],
        },
        {
          label: "Reference",
          items: [
            { label: "Project config", slug: "reference/project-config" },
            { label: "Runner protocol", slug: "reference/runner-protocol" },
            { label: "Release", slug: "reference/release" },
            { label: "Glossary", slug: "reference/glossary" },
          ],
        },
      ],
    }),
  ],
});
