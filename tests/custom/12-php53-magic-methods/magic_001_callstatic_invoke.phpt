--TEST--
Magic methods 001: __callStatic (NEW in 5.3), __invoke (NEW in 5.3), __call, __get/__set
--FILE--
<?php
class Router {
    public static function __callStatic($name, $args) {
        return "static-call:$name(" . implode(',', $args) . ")";
    }
    public function __call($name, $args) {
        return "instance-call:$name(" . implode(',', $args) . ")";
    }
}
var_dump(Router::get('/home', 'HomeController'));
$r = new Router();
var_dump($r->post('/save'));

class Multiplier {
    private $factor;
    public function __construct($factor) { $this->factor = $factor; }
    public function __invoke($x) {
        return $x * $this->factor;
    }
}
$double = new Multiplier(2);
var_dump($double(21));
var_dump(is_callable($double));
var_dump(call_user_func($double, 10));
var_dump(array_map($double, array(1, 2, 3)));

class Magic {
    private $data = array();
    public function __get($name) {
        return isset($this->data[$name]) ? $this->data[$name] : null;
    }
    public function __set($name, $value) {
        $this->data[$name] = $value;
    }
    public function __isset($name) {
        return isset($this->data[$name]);
    }
    public function __unset($name) {
        unset($this->data[$name]);
    }
    public function __toString() {
        return 'Magic(' . implode(',', array_keys($this->data)) . ')';
    }
}
$m = new Magic();
$m->foo = 'bar';
var_dump($m->foo);
var_dump(isset($m->foo));
unset($m->foo);
var_dump(isset($m->foo));
$m->x = 1;
$m->y = 2;
var_dump((string)$m);
echo "$m\n";
--EXPECT--
string(37) "static-call:get(/home,HomeController)"
string(25) "instance-call:post(/save)"
int(42)
bool(true)
int(20)
array(3) {
  [0]=>
  int(2)
  [1]=>
  int(4)
  [2]=>
  int(6)
}
string(3) "bar"
bool(true)
bool(false)
string(10) "Magic(x,y)"
Magic(x,y)
