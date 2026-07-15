<?php
class AliasSource {
    const LABEL = 'source';
    public static $count = 2;

    public static function bump() {
        self::$count = self::$count + 1;
        return self::$count;
    }

    public function name() {
        return 'object';
    }
}
