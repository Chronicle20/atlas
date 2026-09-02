# Tier 1 Reactor Inventory fixture — structural variants

### `1002008`
*(no comment in source)*
```javascript
function act() {
    rm.dropItems();
}
```

### `1012000`
**Source comment:** @Author Lerk * * 1012000.js: Ellinia Plant - drops meso, tree branches, red pots, and Plant Samples (quest item) www.gnu.org/licenses/>.
```javascript
function act() {
    rm.dropItems(true, 2, 20, 40);
}
```

### `2119000`
**Source comment:** * Tombstone in Forest of Dead Trees I MSEA reference: http://mymapleland.blogspot.com/2009/09/kill-lich-at-forest-of-dead-trees-i-to.html www.gnu.org/licenses/>. mymapleland.blogspot.com/2009/09/kill-lich-at-forest-of-dead-trees-i-to.html If the chest is destroyed before Riche, killing him should yield no exp
```javascript
function hit() {
    if (rm.getReactor().getState() !== 0) {
        return
    }
    rm.weakenAreaBoss(6090000, "As the tombstone lit up and vanished, Lich lost all his magic abilities.")
}
function act() {
}
```

### `2612004`
**Source comment:** 2612004.js - Zenumist crystal *@author Ronan www.gnu.org/licenses/>.
```javascript
function hit() {
    rm.sprayItems();
}
function act() {}
```

### `9018000`
**Source comment:** * * @author BubblesDev * @purpose Flower 1 www.gnu.org/licenses/>.
```javascript
function act() {
}
```

### `2511000`
**Source comment:** 2511000- Reactor for PPQ [Pirate PQ] @author Jvlaple www.gnu.org/licenses/>.
```javascript
function act() {
    var eim = rm.getPlayer().getEventInstance();
    var now = eim.getIntProperty("openedBoxes");
    var nextNum = now + 1;
    eim.setIntProperty("openedBoxes", nextNum);
    rm.spawnMonster(9300109, 3);
    rm.spawnMonster(9300110, 5);
}
```
