import { defineStore } from 'pinia'

export const userStore = defineStore({
  id: 'users',
  state: () => ({
    users: [],
    filter: "",
    pageSize: 10,
    pageOffset: 0,
    pages: [],
    fetching: false,
  }),
  getters: {
    Find: (state) => {
      return (id) => state.users.find((p) => p.Identifier === id)
    },
    Count: (state) => state.users.length,
    FilteredCount: (state) => state.Filtered.length,
    All: (state) => state.users,
    Filtered: (state) => {
      if (!state.filter) {
        return state.users
      }
      return state.users.filter((u) => {
        return u.Firstname.includes(state.filter) || u.Lastname.includes(state.filter) || u.Email.includes(state.filter) || u.Identifier.includes(state.filter)
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
    async fetch() {
      this.fetching = true;
      /*const response = await fetch('/data/new-arrivals.json');
      try {
        const result = await response.json();
        this.interfaces = result.interfaces;
      } catch (err) {
        this.interfaces = [];
        console.error('Error loading interfaces:', err);
        return err;
      }*/
      this.users = [{
        Identifier: "user0",
        Email:"tester@test.de",
        Firstname: "Franz",
        Lastname:"Tester",
        Source: "ldap",
        Peers: 19,
        IsAdmin: true,
        Phone:"+436789456",
        Department:"IT",
        Active: true
      },{
        Identifier: "user1",
        Email:"noinfo@nix.de",
        Firstname: "Simple",
        Lastname:"User",
        Source: "db",
        Peers: 3,
        IsAdmin: false,
        Phone:"",
        Department:"",
        Active: false
      },{
        Identifier: "another unique id",
        Email:"Volodimirius.longemail@test.de",
        Firstname: "Volodimirius",
        Lastname:"Longnameinus",
        Source: "ldap",
        Peers: 200,
        IsAdmin: true,
        Phone:"+12345798564897",
        Department:"Sales Lead",
        Active: true
      }];

      this.fetching = false
      this.calculatePages()
    }
  }
})
