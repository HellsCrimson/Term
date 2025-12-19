import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import wails from "@wailsio/runtime/plugins/vite";
import path from 'path';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    svelte(),
    wails("./bindings"),
  ],
  resolve: {
    alias: {
      '$bindings': path.resolve('./bindings'),
      '$lib': path.resolve('./src/lib')
    }
  }
});
