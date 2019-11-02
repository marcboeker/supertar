<template>
  <div class="wrapper">
    <div class="breadcrumb">
      <ul class="path">
        <li>
          <a href="#" @click="browse('')">Home</a>
        </li>
        <li v-for="(p, idx) in this.cleanPath()" v-bind:key="p">
          <a href="#" @click="browse(fullPath(idx))" v-bind:title="fullPath(idx)">{{p}}</a>
        </li>
      </ul>
    </div>
    <List v-bind:items="this.items" :browse="browse" />
  </div>
</template>

<script>
import List from "./List.vue";
import axios from "axios";

export default {
  name: "Browser",
  data() {
    return {
      path: [],
      items: []
    };
  },
  components: {
    List
  },
  mounted() {
    this.browse("");
    this.$root.$on("browse", path => {
      this.browse(path);
    });
  },
  methods: {
    browse(path) {
      this.path = path.split("/");

      axios
        .get(`http://localhost:1337/api/items/${encodeURI(path)}`)
        .then(response => (this.items = response.data));
    },
    cleanPath() {
      if (!this.path[0]) {
        return [];
      }

      if (this.path[0] === ".") {
        return this.path.slice(1);
      }
      return this.path;
    },
    fullPath(idx) {
      return this.path.slice(0, idx + 1).join("/");
    }
  }
};
</script>

<style scoped lang="scss">
.wrapper {
  flex: 1 1 auto;
  display: block;
  width: calc(100% - 300px);
}

.breadcrumb {
  display: flex;
  background: #f8f8f8;
}

.path {
  display: inline-block;
  margin: 0;
  padding: 0.333333rem 1rem;
  list-style: none;
  background: #f8f8f8;
  border-radius: 3px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  line-height: 2;
  border-radius: 0 0 3px 3px;

  li {
    display: inline-block;

    &::after {
      display: inline-block;
      content: "›";
      opacity: 0.5;
      margin-left: 0.5rem;
      margin-right: 0.25rem;
    }

    &:last-child::after {
      content: "";
    }
  }

  a {
    cursor: pointer;

    &:hover {
      color: darken(#787878, 40%);
    }
  }
}
</style>
