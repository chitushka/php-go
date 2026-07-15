--TEST--
P2 magic __sleep and __wakeup
--FILE--
<?php
class User {
    public $name = 'Ada';
    public $secret = 'hidden';
    public $ready = 'no';

    public function __sleep() {
        return array('name');
    }

    public function __wakeup() {
        $this->ready = 'yes';
    }
}

$user = new User();
$encoded = serialize($user);
echo $encoded;
echo "\n";
$copy = unserialize($encoded);
echo $copy->name;
echo "\n";
echo isset($copy->secret) ? 'secret' : 'no-secret';
echo "\n";
echo $copy->ready;
?>
--EXPECT--
O:4:"User":1:{s:4:"name";s:3:"Ada";}
Ada
no-secret
yes
