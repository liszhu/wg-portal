import { defineStore } from 'pinia'

import {apiWrapper} from '../helpers/fetch-wrapper.js'
import {notify} from "@kyvg/vue3-notification";

export const interfaceStore = defineStore({
  id: 'interfaces',
  state: () => ({
    interfaces: [],
    selected: "wg0",
    fetching: false,
  }),
  getters: {
    Count: (state) => state.interfaces.length,
    All: (state) => state.interfaces,
    GetSelected: (state) => state.interfaces.find((i) => i.Identifier === state.selected) || state.interfaces[0],
  },
  actions: {
    async LoadInterfaces() {
      apiWrapper.get(`/interfaces`)
          .then(interfaces => this.interfaces = interfaces)
          .catch(error => {
            this.interfaces = []
            console.log("Failed to load interfaces: ", error)
            notify({
              title: "Backend Connection Failure",
              text: "Failed to load interfaces!",
            })
          })
    }
  }
})
