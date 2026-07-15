<?php
interface ParentContract {
}

interface ChildContract extends ParentContract {
}

class HierBase {
}

class HierChild extends HierBase implements ChildContract {
    public function parentName() {
        return get_parent_class();
    }
}
