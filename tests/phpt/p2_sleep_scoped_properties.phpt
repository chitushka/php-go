--TEST--
P2 __sleep resolves scoped property names
--FILE--
<?php
class SleepScopedBase {
    protected $shared = 'prot';
}

class SleepScopedChild extends SleepScopedBase {
    private $childSecret = 'child';
    public $name = 'Ada';

    public function __sleep() {
        return array('name', 'shared', 'childSecret', 'missing');
    }

    public function read() {
        return $this->name . ':' . $this->shared . ':' . $this->childSecret . ':' . (isset($this->missing) ? 'missing' : 'no');
    }
}

$encoded = serialize(new SleepScopedChild());
$copy = unserialize($encoded);
echo $copy->read();
?>
--EXPECT--
Ada:prot:child:no
