import { defineStore } from 'pinia'
import {apiWrapper} from "../helpers/fetch-wrapper";
import {notify} from "@kyvg/vue3-notification";
import {interfaceStore} from "./interfaces";

export const peerStore = defineStore({
  id: 'peers',
  state: () => ({
    peers: [],
    filter: "",
    pageSize: 10,
    pageOffset: 0,
    pages: [],
    fetching: false,
  }),
  getters: {
    Find: (state) => {
      return (id) => state.peers.find((p) => p.Identifier === id)
    },
    Count: (state) => state.peers.length,
    FilteredCount: (state) => state.Filtered.length,
    All: (state) => state.peers,
    Filtered: (state) => {
      if (!state.filter) {
        return state.peers
      }
      return state.peers.filter((p) => {
        return p.Name.includes(state.filter) || p.Identifier.includes(state.filter)
      })
    },
    FilteredAndPaged: (state) => {
      return state.Filtered.slice(state.pageOffset, state.pageOffset + state.pageSize)
    },
    isFetching: (state) => state.fetching,
    hasNextPage: (state) => state.pageOffset < (state.FilteredCount - state.pageSize),
    hasPrevPage: (state) => state.pageOffset > 0,
    currentPage: (state) => (state.pageOffset / state.pageSize)+1,
  },
  actions: {
    afterPageSizeChange() {
      // reset pageOffset to avoid problems with new page sizes
      this.pageOffset = 0
      this.calculatePages()
    },
    calculatePages() {
      let pageCounter = 1;
      this.pages = []
      for (let i = 0; i < this.FilteredCount; i+=this.pageSize) {
        this.pages.push(pageCounter++)
      }
    },
    gotoPage(page) {
      this.pageOffset = (page-1) * this.pageSize

      this.calculatePages()
    },
    nextPage() {
      this.pageOffset += this.pageSize

      this.calculatePages()
    },
    previousPage() {
      this.pageOffset -= this.pageSize

      this.calculatePages()
    },
    async LoadSomePeers(offset, pageSize) {
      apiWrapper.get(`/peers` + query)
          .then(peers => {
            this.peers = peers
            this.calculatePages()
          })
          .catch(error => {
            this.peers = []
            console.log("Failed to load peers: ", error)
            notify({
              title: "Backend Connection Failure",
              text: "Failed to load peers!",
            })
          })
    },
    async LoadPeers() {
      this.peers = [] // reset peers

      const pageSize = 50 // load 50 peers at a time

      let offset = 0
      let fetching = true
      let result = {}
      let allResults = []

      let iface = interfaceStore().GetSelected
      if (!iface) {
        return // no interface, nothing to load
      }

      while(fetching) {
        if(offset === 0 || result.MoreRecords === true) {
          let query = "?offset=" + offset + "&pagesize=" + pageSize + "&interface=" + iface.Identifier
          try {
            result = await apiWrapper.get(`/peers` + query)
            console.log("RESPONSE:", result)
            console.log("RESPONSE RECS:", result.Records)
          } catch (e) {
            console.log("peer load error")
            return Promise.reject(e.Message)
          }
          offset += pageSize
        } else if(result.MoreRecords === false) {
          fetching = false
        }

        allResults = allResults.concat(result.Records)
      }
      this.peers = allResults
    }
  }
})
