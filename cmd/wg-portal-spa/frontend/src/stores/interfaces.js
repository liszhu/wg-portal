import { defineStore } from 'pinia'

import {apiWrapper} from '../helpers/fetch-wrapper.js'
import {notify} from "@kyvg/vue3-notification";

export const interfaceStore = defineStore({
  id: 'interfaces',
  state: () => ({
    interfaces: [],
    prepared: {
        Identifier: "",
        Type: "server",
    },
    selected: "wg0",
    fetching: false,
  }),
  getters: {
    Count: (state) => state.interfaces.length,
    Prepared: (state) => {console.log("STATE:", state.prepared); return state.prepared},
    All: (state) => state.interfaces,
    Find: (state) => {
        return (id) => state.interfaces.find((p) => p.Identifier === id)
    },
    GetSelected: (state) => state.interfaces.find((i) => i.Identifier === state.selected) || state.interfaces[0],
  },
  actions: {
    setInterfaces(interfaces) {
      this.interfaces = interfaces;
    },
    async LoadInterfaces() {
      return apiWrapper.get(`/interfaces`)
          .then(this.setInterfaces)
          .catch(error => {
            this.interfaces = []
            console.log("Failed to load interfaces: ", error)
            notify({
              title: "Backend Connection Failure",
              text: "Failed to load interfaces!",
            })
          })
    },
    setPreparedInterface(iface) {
      this.prepared = iface;
    },
    async PrepareInterface() {
      return apiWrapper.get(`/interfaces/prepare`)
        .then(this.setPreparedInterface)
        .catch(error => {
          this.prepared = {}
          console.log("Failed to load prepared interface: ", error)
          notify({
            title: "Backend Connection Failure",
            text: "Failed to load prepared interface!",
          })
        })
  }
  }
})
