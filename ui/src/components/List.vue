<template>
  <table class="list">
    <tr class="list__header">
      <th class="list__header__name">Name</th>
      <th class="list__header__size">Size</th>
      <th class="list__header__mtime">Last modification</th>
    </tr>
    <tr
      v-if="item.isDir"
      class="list__item--directory"
      v-for="(item, idx) in this.items"
      @click="browse(`${item.path}/${item.name}`)"
    >
      <td>{{item.name}}</td>
      <td>-</td>
      <td>{{item.mtime}}</td>
    </tr>
    <tr
      v-if="!item.isDir"
      class="list__item--file"
      v-for="(item, idx) in this.items"
      @click="stream(`${item.path}/${item.name}`)"
    >
      <td>{{item.name}}</td>
      <td>{{item.size}}</td>
      <td>{{item.mtime}}</td>
    </tr>
  </table>
</template>

<script>
export default {
  name: "List",
  props: {
    items: Array
  },
  methods: {
    stream(path) {
      window.open(
        `http://localhost:1337/api/stream/${encodeURI(path)}`,
        "_blank"
      );
    },
    browse(path) {
      this.$root.$emit("browse", path);
    }
  }
};
</script>

<style scoped lang="scss">
.list {
  width: 100%;
  text-align: left;
  font-size: 0.875rem;

  th {
    user-select: none;
    border-bottom: 1px solid #efefef;
    padding: 1rem;
    background: #fff;

    &:hover {
      cursor: pointer;
      color: darken(#787878, 40%);
    }
  }

  tr:nth-child(odd) {
    background: #f8f8f8;
  }

  td {
    padding: 0.5rem 1rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;

    &:hover {
      cursor: pointer;
      color: darken(#787878, 40%);
    }

    &:first-child {
      max-width: 200px;
    }
  }
}

.list__item--directory {
  td:first-child {
    font-weight: bold;
  }
}

.list__item--file {
}

.list__header {
  font-size: 0.6875rem;
  letter-spacing: 1px;
  line-height: 1;
}

.list__header__name {
}

.list__header__size {
  width: 100px;
}

.list__header__mtime {
  width: 220px;
}
</style>


